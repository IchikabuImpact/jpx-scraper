package main

import (
    "database/sql"
    "fmt"
    "log"
    "net/http"

    "github.com/IchikabuImpact/jpx-scraper/internal/stockdata"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    _ "github.com/mattn/go-sqlite3"
)

func main() {
    e := echo.New()

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())

    db, err := sql.Open("sqlite3", "./stockdata.db")
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()

    // テーブル作成
    createTableSQL := `
    CREATE TABLE IF NOT EXISTS scrapings (
        ticker TEXT PRIMARY KEY,
        jsond TEXT,
        updated TIMESTAMP
    );
    `
    if _, err := db.Exec(createTableSQL); err != nil {
        log.Fatalf("Failed to create table: %v", err)
    }

    e.GET("/scrape", func(c echo.Context) error {
        ticker := c.QueryParam("ticker")
        if ticker == "" {
            return c.JSON(http.StatusBadRequest, map[string]string{"error": "Ticker is required"})
        }

        data, err := stockdata.GetStockDataJSON(ticker, db)
        if err != nil {
            return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Error retrieving stock data: %v", err)})
        }

        return c.JSONBlob(http.StatusOK, []byte(data))
    })

    log.Fatal(e.Start(":8081"))
}

