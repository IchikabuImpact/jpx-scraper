package stockdata

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// StockData represents the stock data structure
type StockData struct {
	Ticker        string `json:"ticker"`
	CompanyName   string `json:"companyName"`
	CurrentPrice  string `json:"currentPrice"`
	PreviousClose string `json:"previousClose"`
	DividendYield string `json:"dividendYield"`
	PER           string `json:"per,omitempty"`
	PBR           string `json:"pbr,omitempty"`
	MarketCap     string `json:"marketCap,omitempty"`
	Volume        string `json:"volume,omitempty"`
	Open          string `json:"open,omitempty"`
	High          string `json:"high,omitempty"`
	Low           string `json:"low,omitempty"`
	Close         string `json:"close,omitempty"`
}

func trimDisplaySuffix(value string, suffixes ...string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, "\u00a0", " "))
	for _, suffix := range suffixes {
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, suffix))
	}
	return trimmed
}

// ohlcFromKobetsuTable reads the open/high/low/close table in #kobetsu_left,
// matching rows by their <th> label text so it stays correct even if row order changes.
func ohlcFromKobetsuTable(doc *goquery.Document) (open, high, low, close string) {
	// Label constants match Kabutan's Japanese row headers exactly (open/high/low/close).
	const (
		labelOpen  = "\u59cb\u5024"
		labelHigh  = "\u9ad8\u5024"
		labelLow   = "\u5b89\u5024"
		labelClose = "\u7d42\u5024"
	)
	// Direct-child combinator matters here: the PTS (after-hours) section further down
	// #kobetsu_left also has a table with the same 始値/高値/安値 <th> labels, filled with
	// "－" placeholders for tickers with no PTS trading. A plain descendant selector matches
	// that nested table too (it's table #1 within its own parent, div.stock_pts_div), and
	// since it comes later in the document its placeholders silently overwrite the real values.
	doc.Find("#kobetsu_left > table:nth-of-type(1) tbody tr").Each(func(_ int, row *goquery.Selection) {
		label := strings.TrimSpace(row.Find("th").First().Text())
		value := strings.TrimSpace(row.Find("td").First().Text())
		switch label {
		case labelOpen:
			open = value
		case labelHigh:
			high = value
		case labelLow:
			low = value
		case labelClose:
			close = value
		}
	})
	return open, high, low, close
}

// ValidateTicker checks if the ticker is valid (only contains letters and numbers)
func ValidateTicker(ticker string) error {
	validTicker := regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString
	if !validTicker(ticker) {
		return errors.New("invalid ticker: ticker should only contain letters and numbers")
	}
	return nil
}

// Function to get stock data from an external API
func GetStockData(ticker string) (StockData, error) {
	// Validate ticker before proceeding
	if err := ValidateTicker(ticker); err != nil {
		return StockData{}, err
	}

	const (
		KabutanURL = "https://kabutan.jp/stock/?code=%s"
	)

	url := fmt.Sprintf(KabutanURL, ticker)

	time.Sleep(400 * time.Millisecond)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return StockData{}, fmt.Errorf("failed to build request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")
	req.Header.Set("Referer", "https://kabutan.jp/")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return StockData{}, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		parsed, parseErr := urlpkg.Parse(url)
		host := ""
		if parseErr == nil {
			host = parsed.Host
		}
		return StockData{}, fmt.Errorf("unexpected status code: %d from %s", resp.StatusCode, host)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return StockData{}, fmt.Errorf("failed to parse document: %v", err)
	}

	companyName := doc.Find(".si_i1_1 h2").Text()
	if companyName == "" {
		return StockData{}, fmt.Errorf("failed to find company name")
	}

	currentPrice := trimDisplaySuffix(doc.Find(".si_i1_2 .kabuka").Text(), "円")
	previousClose := strings.TrimSpace(doc.Find("#kobetsu_left dl dd").First().Text())
	open, high, low, closePrice := ohlcFromKobetsuTable(doc)
	dividendYield := trimDisplaySuffix(doc.Find("#stockinfo_i3 tbody tr:nth-child(1) td:nth-child(3)").Text(), "％")
	per := trimDisplaySuffix(doc.Find("#stockinfo_i3 tbody tr:nth-child(1) td:nth-child(1)").Text(), "倍") // PER
	pbr := trimDisplaySuffix(doc.Find("#stockinfo_i3 tbody tr:nth-child(1) td:nth-child(2)").Text(), "倍") // PBR
	marketCap := strings.TrimSpace(doc.Find("#stockinfo_i3 tbody tr:nth-child(2) td").First().Text())     // 時価総額
	volumeRaw := strings.TrimSpace(doc.Find("#kobetsu_left table:nth-of-type(2) tbody tr:nth-child(1) td").First().Text())
	if volumeRaw == "" {
		volumeRaw = strings.TrimSpace(doc.Find("body div:nth-child(1) div:nth-child(3) div:nth-child(1) div:nth-child(3) table:nth-of-type(2) tbody tr:nth-child(1) td").First().Text())
	}
	volume := strings.TrimSpace(strings.ReplaceAll(volumeRaw, "\u00a0", " "))
	volume = strings.TrimSpace(strings.TrimSuffix(volume, "株"))
	return StockData{
		Ticker:        ticker,
		CompanyName:   companyName,
		CurrentPrice:  currentPrice,
		PreviousClose: previousClose,
		DividendYield: dividendYield,
		PER:           per,
		PBR:           pbr,
		MarketCap:     marketCap,
		Volume:        volume,
		Open:          open,
		High:          high,
		Low:           low,
		Close:         closePrice,
	}, nil
}

func GetStockDataJSON(ticker string, db *sql.DB) (string, error) {
	var jsonData string
	var updated time.Time

	query := "SELECT jsond, updated FROM scrapings WHERE ticker = ?"
	err := db.QueryRow(query, ticker).Scan(&jsonData, &updated)
	if err == nil {
		// データが存在し、1時間以内の場合キャッシュを返す
		if time.Since(updated) < time.Hour {
			return jsonData, nil
		}
	}

	// データがないか、1時間以上経過している場合は新たにスクレイピング
	data, err := GetStockData(ticker)
	if err != nil {
		fmt.Printf("Error in GetStockData: %v\n", err)
		return "", err
	}

	jsonDataBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		return "", err
	}
	jsonData = string(jsonDataBytes)

	// 非同期でデータベースに保存
	go func() {
		insertQuery := `
        REPLACE INTO scrapings (ticker, jsond, updated)
        VALUES (?, ?, ?)
        `
		_, err := db.Exec(insertQuery, data.Ticker, jsonData, time.Now().UTC())
		if err != nil {
			fmt.Printf("Error inserting data into database: %v\n", err)
		}
	}()

	return jsonData, nil
}
