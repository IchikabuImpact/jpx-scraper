package stockdata

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "github.com/tebeka/selenium"
    "regexp"
    "time"
    "errors"
)

// 定数を定義
const (
    seleniumURL      = "http://selenium-hub:4445/wd/hub"
    urlGoogleFinance = "https://www.google.com/finance/quote/%s:TYO?hl=ja"
    companyNameSelector = ".zzDege"
    currentPriceSelector = ".YMlKec.fxKbKc"
    previousCloseSelector = "div.P6K39c"
)

// StockData represents the stock data structure
type StockData struct {
    Ticker        string `json:"ticker"`
    CompanyName   string `json:"companyName"`
    CurrentPrice  string `json:"currentPrice"`
    PreviousClose string `json:"previousClose"`
}

// ValidateTicker checks if the ticker is valid (only contains letters and numbers)
func ValidateTicker(ticker string) error {
    validTicker := regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString
    if !validTicker(ticker) {
        return errors.New("invalid ticker: ticker should only contain letters and numbers")
    }
    return nil
}

// Function to get stock data from Google Finance
func GetStockData(ticker string) (StockData, error) {
    // Validate ticker before proceeding
    if err := ValidateTicker(ticker); err != nil {
        return StockData{}, err
    }

    caps := selenium.Capabilities{
        "browserName": "chrome",
        "goog:chromeOptions": map[string]interface{}{
            "args": []string{"--headless", "--disable-cache"},
        },
    }

    wd, err := selenium.NewRemote(caps, seleniumURL)
    if err != nil {
        fmt.Printf("Error creating new WebDriver: %v\n", err)
        return StockData{}, fmt.Errorf("error creating new WebDriver: %v", err)
    }
    defer wd.Quit()

    // Get stock data from Google Finance
    err = wd.Get(fmt.Sprintf(urlGoogleFinance, ticker))
    if err != nil {
        fmt.Printf("Error loading Google Finance page: %v\n", err)
        return StockData{}, fmt.Errorf("error loading Google Finance page: %v", err)
    }

    companyName, err := getElementText(wd, companyNameSelector)
    if err != nil {
        fmt.Printf("Error getting company name: %v\n", err)
        return StockData{}, fmt.Errorf("error getting company name: %v", err)
    }

    currentPrice, err := getElementText(wd, currentPriceSelector)
    if err != nil {
        fmt.Printf("Error getting current price: %v\n", err)
        return StockData{}, fmt.Errorf("error getting current price: %v", err)
    }

    previousClose, err := getElementText(wd, previousCloseSelector)
    if err != nil {
        fmt.Printf("Error getting previous close: %v\n", err)
        return StockData{}, fmt.Errorf("error getting previous close: %v", err)
    }

    return StockData{
        Ticker:        ticker,
        CompanyName:   companyName,
        CurrentPrice:  currentPrice,
        PreviousClose: previousClose,
    }, nil
}

func getElementText(wd selenium.WebDriver, value string) (string, error) {
    elem, err := wd.FindElement(selenium.ByCSSSelector, value)
    if err != nil {
        return "", err
    }
    text, err := elem.Text()
    if err != nil {
        return "", err
    }
    return text, nil
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
        _, err := db.Exec(insertQuery, data.Ticker, jsonData, time.Now())
        if err != nil {
            fmt.Printf("Error inserting data into database: %v\n", err)
        }
    }()

    return jsonData, nil
}

