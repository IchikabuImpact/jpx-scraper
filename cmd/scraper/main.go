package main

import (
    "database/sql"
    "fmt"
    "log"
    "net/http"

    "github.com/gin-gonic/gin"
    _ "github.com/mattn/go-sqlite3"

    "github.com/IchikabuImpact/jpx-scraper/internal/stockdata"
)

func main() {
    // Gin のルーターを作成
    r := gin.New()
    r.Use(gin.Logger())
    r.Use(gin.Recovery())

    // SQLite オープン
    db, err := sql.Open("sqlite3", "./stockdata.db")
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()

    // テーブル作成（初回のみ実行）
    const createTableSQL = `
        CREATE TABLE IF NOT EXISTS scrapings (
            ticker  TEXT PRIMARY KEY,
            jsond   TEXT,
            updated TIMESTAMP
        );`
    if _, err := db.Exec(createTableSQL); err != nil {
        log.Fatalf("Failed to create table: %v", err)
    }

    // /scrape?ticker=1332 のルート
    r.GET("/scrape", func(c *gin.Context) {
        ticker := c.Query("ticker")
        if ticker == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker is required"})
            return
        }

        data, err := stockdata.GetStockDataJSON(ticker, db)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": fmt.Sprintf("Error retrieving stock data: %v", err),
            })
            return
        }

        c.Data(http.StatusOK, "application/json", []byte(data))
    })

    // ポート 8081 で待受（Docker 設定と合わせる）
    if err := r.Run(":8081"); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
