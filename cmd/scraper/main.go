// cmd/stockdata/main.go
package main

import (
    "fmt"
    "log"
    "net/http"
    "fbstocks/internal/stockdata"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    e := echo.New()

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())

    e.GET("/scrape", func(c echo.Context) error {
        ticker := c.QueryParam("ticker")
        if ticker == "" {
            return c.JSON(http.StatusBadRequest, map[string]string{"error": "Ticker is required"})
        }

        data, err := stockdata.GetStockDataJSON(ticker)
        if err != nil {
            return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Error retrieving stock data: %v", err)})
        }

        return c.JSONBlob(http.StatusOK, []byte(data))
    })

    log.Fatal(e.Start(":8081"))
}

