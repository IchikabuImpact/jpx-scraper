# JPX Scraper

JPX Scraper is a Go/Gin API that scrapes stock data from Kabutan, caches the JSON in MariaDB, and exposes it through `/scrape?ticker=XXXX`. The application is built to run inside Docker along with Selenium Grid for browser automation.

## Architecture

```text
                          +--------------------------+
                          |    Reverse Proxy (opt)   |
                          |    :443 -> :8082         |
                          +------------+-------------+
                                       |
                                       v
+--------------------+    +------------+-------------+    +----------------------+
| Selenium Node      |<-->|   Selenium Hub (4444)    |<-->|   Selenium Clients   |
| Chrome (1 session) |    +------------+-------------+    +----------------------+
+---------+----------+                 |
          |                            |
          |                            v
          |               +------------+-------------+
          +-------------->|     JPX Scraper API      |
                          |   Go HTTP :8081 (host :8082)
                          |   Waits for DB & Selenium |
                          +------------+-------------+
                                       |
                                       v
                          +------------+-------------+
                          |    MariaDB 11.x (3306)   |
                          |   scrapings cache table  |
                          +--------------------------+
```

All containers share a single Docker bridge network created by `docker compose` so services can communicate by name (`mariadb`, `selenium-hub`, `selenium-node`, `jpx-scraper`).

## Getting Started

### 1. Prepare environment variables

```bash
cp .env.example .env
# Edit .env and set strong passwords for MariaDB and the scraper user.
```

> **Security note:** Never commit `.env` to source control. The application and Docker Compose read secrets from environment variables at runtime only.

### 2. Launch the stack with Docker Compose

```bash
docker compose --env-file .env up -d --build
```

This command builds the Go scraper image from `Dockerfile.scraper`, starts MariaDB (with health checks), Selenium Hub/Node, and the scraper API. The scraper waits for MariaDB to become reachable, applies the `scrapings` table schema if needed, and then begins serving traffic on container port `8081` (published to `http://localhost:8082`).

### 3. Verify the API

```bash
curl "http://localhost:8082/scrape?ticker=1332"
```

A JSON payload should be returned. MariaDB caches responses for one hour, so repeat requests will hit the database instead of re-scraping until the cache expires.

## Manual build & run fallback

If you cannot use Docker Compose, you can run each component manually. Ensure you export the same variables defined in `.env` before running these commands.

```bash
# Build images
docker build -t jpx-scraper -f Dockerfile.scraper .
docker build -t selenium-hub-custom -f Dockerfile.selenium .

# Create a network
docker network create jpx-stack 2>/dev/null || true

# MariaDB
docker run -d --name mariadb \
  --env-file .env \
  --network jpx-stack \
  -p 3306:3306 \
  -v mariadb_data:/var/lib/mysql \
  mariadb:11.4

# Selenium Hub & Node
docker run -d --name selenium-hub \
  --network jpx-stack \
  -p 4444:4444 \
  selenium-hub-custom

docker run -d --name selenium-node \
  --network jpx-stack \
  -e SE_EVENT_BUS_HOST=selenium-hub \
  -e SE_EVENT_BUS_PUBLISH_PORT=4442 \
  -e SE_EVENT_BUS_SUBSCRIBE_PORT=4443 \
  -e SE_NODE_MAX_SESSIONS=1 \
  selenium/node-chrome:latest

# Scraper API
docker run -d --name jpx-scraper \
  --network jpx-stack \
  --env-file .env \
  -p 8082:8081 \
  jpx-scraper
```

When you are finished, stop everything with `docker compose down` (or stop the individual containers created manually).

## Ports & health checks

| Service        | Host Port | Container Port | Health Check                                  |
|----------------|-----------|----------------|-----------------------------------------------|
| MariaDB        | 3306      | 3306           | `mysqladmin ping` inside the container        |
| Selenium Hub   | 4444      | 4444           | `http://localhost:4444/status`                |
| Selenium Node  | (n/a)     | internal only  | Depends on Selenium Hub health                |
| JPX Scraper    | 8082      | 8081           | Waits on MariaDB/Selenium health before start |

## Logs & troubleshooting

- Follow container logs: `docker compose logs -f mariadb` (or `selenium-hub`, `selenium-node`, `scraper`).
- Open a shell inside the scraper: `docker compose exec scraper sh`.
- Inspect MariaDB: `docker compose exec mariadb mariadb -u "$MARIADB_USER" -p"$MARIADB_PASSWORD" "$MARIADB_DATABASE"`.
- Test service reachability: `curl http://localhost:4444/status` for Selenium Hub, `curl http://localhost:8082/scrape?ticker=1332` for the API.
- Rebuild after code changes: `docker compose --env-file .env up -d --build`.

If the scraper reports database connection errors, confirm MariaDB is healthy (`docker compose ps`), credentials in `.env` match the Compose configuration, and no firewall is blocking port `3306` locally.

## Local development

You can run Go tests and builds directly on your workstation without Docker:

```bash
go test ./...
go build ./...
```

The scraper builds a MariaDB DSN from environment variables, waits for the database to be ready using `PingContext`, ensures the `scrapings` table exists (idempotently), and configures connection pooling (`DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`).

## Contributing

1. Fork the repository and create a feature branch.
2. Ensure `go test ./...` passes and linting is clean.
3. Submit a pull request describing your changes.

## License

This project is licensed under the MIT License.
