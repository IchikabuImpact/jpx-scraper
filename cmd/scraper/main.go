package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"github.com/IchikabuImpact/jpx-scraper/internal/stockdata"
)

const (
	defaultDBHost          = "mariadb"
	defaultDBPort          = "3306"
	defaultDBName          = "jpx"
	defaultDBUser          = "jpx"
	defaultDBParams        = "parseTime=true&charset=utf8mb4"
	defaultMaxOpenConns    = 10
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = time.Minute * 5
	defaultDBWaitTimeout   = time.Minute
	defaultDBPingInterval  = time.Second
	httpListenAddr         = ":8081"
)

func main() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	db, err := openDBFromEnv()
	if err != nil {
		log.Fatalf("failed to configure database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getDBWaitTimeout())
	defer cancel()

	if err := waitForDB(ctx, db, defaultDBPingInterval); err != nil {
		log.Fatalf("database not ready: %v", err)
	}

	if err := ensureSchema(db); err != nil {
		log.Fatalf("failed to ensure schema: %v", err)
	}

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

	if err := r.Run(httpListenAddr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func openDBFromEnv() (*sql.DB, error) {
	host := getEnvOrDefault("DB_HOST", defaultDBHost)
	port := getEnvOrDefault("DB_PORT", defaultDBPort)
	name := getEnvOrDefault("DB_NAME", defaultDBName)
	user := getEnvOrDefault("DB_USER", defaultDBUser)
	password := os.Getenv("DB_PASSWORD")
	params := getEnvOrDefault("DB_PARAMS", defaultDBParams)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, name)
	if params != "" {
		dsn = fmt.Sprintf("%s?%s", dsn, params)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	maxOpenConns, err := getIntFromEnv("DB_MAX_OPEN_CONNS", defaultMaxOpenConns)
	if err != nil {
		return nil, err
	}
	if maxOpenConns > 0 {
		db.SetMaxOpenConns(maxOpenConns)
	}

	maxIdleConns, err := getIntFromEnv("DB_MAX_IDLE_CONNS", defaultMaxIdleConns)
	if err != nil {
		return nil, err
	}
	if maxIdleConns >= 0 {
		db.SetMaxIdleConns(maxIdleConns)
	}

	connLifetime, err := getDurationFromEnv("DB_CONN_MAX_LIFETIME", defaultConnMaxLifetime)
	if err != nil {
		return nil, err
	}
	if connLifetime > 0 {
		db.SetConnMaxLifetime(connLifetime)
	}

	return db, nil
}

func waitForDB(ctx context.Context, db *sql.DB, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastErr error

	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		} else {
			lastErr = err
			log.Printf("waiting for database to become ready: %v", err)
		}

		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				if lastErr != nil {
					return fmt.Errorf("timed out waiting for database: %w (last error: %v)", ctx.Err(), lastErr)
				}
				return fmt.Errorf("timed out waiting for database: %w", ctx.Err())
			}
			if lastErr != nil {
				return fmt.Errorf("context closed while waiting for database: %v", lastErr)
			}
			return fmt.Errorf("context closed while waiting for database")
		case <-ticker.C:
		}
	}
}

func ensureSchema(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS scrapings (
    ticker  VARCHAR(32) NOT NULL PRIMARY KEY,
    jsond   LONGTEXT,
    updated DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("create scrapings table: %w", err)
	}
	return nil
}

func getEnvOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func getIntFromEnv(key string, def int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %w", key, err)
	}

	return val, nil
}

func getDurationFromEnv(key string, def time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}

	return d, nil
}

func getDBWaitTimeout() time.Duration {
	timeout, err := getDurationFromEnv("DB_WAIT_TIMEOUT", defaultDBWaitTimeout)
	if err != nil {
		log.Printf("invalid DB_WAIT_TIMEOUT: %v, falling back to default %s", err, defaultDBWaitTimeout)
		return defaultDBWaitTimeout
	}
	return timeout
}
