package stockdata

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	resp, err := http.Get(url)
	if err != nil {
		return StockData{}, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StockData{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return StockData{}, fmt.Errorf("failed to parse document: %v", err)
	}

	companyName := doc.Find(".si_i1_1 h2").Text()
	if companyName == "" {
		return StockData{}, fmt.Errorf("failed to find company name")
	}

	currentPrice := strings.TrimSpace(doc.Find(".si_i1_2 .kabuka").Text())
	previousClose := strings.TrimSpace(doc.Find("#kobetsu_left dl dd").First().Text())
	dividendYield := strings.TrimSpace(doc.Find("#stockinfo_i3 tbody tr:nth-child(1) td:nth-child(3)").Text())
	per := strings.TrimSpace(doc.Find("#stockinfo_i3 tbody tr:nth-child(1) td:nth-child(1)").Text())  // PER
	pbr := strings.TrimSpace(doc.Find("#stockinfo_i3 tbody tr:nth-child(1) td:nth-child(2)").Text())  // PBR
	marketCap := strings.TrimSpace(doc.Find("#stockinfo_i3 tbody tr:nth-child(2) td").First().Text()) // 時価総額
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
