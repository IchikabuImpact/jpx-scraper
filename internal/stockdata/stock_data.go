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

// StockData represents the stock data structure
type StockData struct {
    Ticker        string `json:"ticker"`
    CompanyName   string `json:"companyName"`
    CurrentPrice  string `json:"currentPrice"`
    PreviousClose string `json:"previousClose"`
//    DividendYield string `json:"dividendYield"` // 追加
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
        seleniumURL      = "http://selenium-hub:4445/wd/hub"
        urlGoogleFinance = "https://www.google.com/finance/quote/%s:TYO?hl=ja"
    )

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

    companyName, err := getElementText(wd, ".zzDege")
    if err != nil {
        fmt.Printf("Error getting company name: %v\n", err)
        return StockData{}, fmt.Errorf("error getting company name: %v", err)
    }

    currentPrice, err := getElementText(wd, ".YMlKec.fxKbKc")
    if err != nil {
        fmt.Printf("Error getting current price: %v\n", err)
        return StockData{}, fmt.Errorf("error getting current price: %v", err)
    }

    previousClose, err := getElementText(wd, "div.P6K39c")
    if err != nil {
        fmt.Printf("Error getting previous close: %v\n", err)
        return StockData{}, fmt.Errorf("error getting previous close: %v", err)
    }

    // 配当利回りの取得
/*
    dividendYield, err := getDividendYield(wd)
    if err != nil {
        fmt.Printf("Error getting dividend yield: %v\n", err)
        return StockData{}, fmt.Errorf("error getting dividend yield: %v", err)
    }
*/

    return StockData{
        Ticker:        ticker,
        CompanyName:   companyName,
        CurrentPrice:  currentPrice,
        PreviousClose: previousClose,
  //      DividendYield: dividendYield,
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
/*
func getDividendYield(wd selenium.WebDriver) (string, error) {
    xpath := "//div[@aria-describedby='c533']/div[@class='P6K39c']"
    elem, err := wd.FindElement(selenium.ByXPATH, xpath)
    if err != nil {
        return "", err
    }
    text, err := elem.Text()
    if err != nil {
        return "", err
    }
    return text, nil
}
*/

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

