package stockdata

import (
    "encoding/json"
    "fmt"
    "github.com/tebeka/selenium"
)

// StockData represents the stock data structure
type StockData struct {
    Ticker        string `json:"ticker"`
    CompanyName   string `json:"companyName"`
    CurrentPrice  string `json:"currentPrice"`
    PreviousClose string `json:"previousClose"`
}

// Function to get stock data from an external API
func GetStockData(ticker string) (StockData, error) {
    const (
        seleniumURL      = "http://selenium-hub:4445/wd/hub"
        urlGoogleFinance = "https://www.google.com/finance/quote/%s:TYO?sa=X&ved=2ahUKEwiG1vL6yZzxAhUD4zgGHQGxD7QQ3ecFegQINRAS"
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

func GetStockDataJSON(ticker string) (string, error) {
    data, err := GetStockData(ticker)
    if err != nil {
        fmt.Printf("Error in GetStockData: %v\n", err)
        return "", err
    }
    jsonData, err := json.Marshal(data)
    if err != nil {
        fmt.Printf("Error marshalling JSON: %v\n", err)
        return "", err
    }
    return string(jsonData), nil
}

