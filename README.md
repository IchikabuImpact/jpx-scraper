# JPX Scraper — Compact Architecture & Ops Guide (develop)

JPX Scraper is a Go API that scrapes stock data (Kabutan), **caches the JSON in MariaDB**, and serves it via HTTP:
```
GET /scrape?ticker=8306
```
This README focuses on **what gets created in MariaDB**, **how to verify with SQL**, and **which knobs control cache lifetime**. It also documents the **compact Docker** profile you’re currently running.

---

## High-level architecture (current)

```
Browser/Client
    │
    ▼
HTTP (REST, JSON)
:8082 → container :8081  ───────┐
                                │ calls
                                ▼
          Scraper (Go) ── DB cache (MariaDB 11.x)
                    └─> table: scrapings (1 row per ticker, JSON payload)
```

> gRPC & KVS(Valkey) are **not enabled yet**. The design anticipates adding them later with minimal changes.

---

## What tables are created? (DDL)

At startup, the app **idempotently ensures** the cache table exists:

```sql
CREATE TABLE IF NOT EXISTS scrapings (
  ticker  VARCHAR(32)  NOT NULL PRIMARY KEY, -- JPX code, e.g. '8306'
  jsond   LONGTEXT     NULL,                 -- raw JSON payload served by /scrape
  updated DATETIME     NOT NULL              -- last refresh timestamp
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- **Primary key**: `ticker` — there is **one row per ticker**. New data **replaces** the existing row.
- **Write path**: when the scraper refreshes data, it uses:
  ```sql
  REPLACE INTO scrapings(ticker, jsond, updated) VALUES(?,?,NOW());
  ```
- **Read path**: the handler first tries the DB row; if **fresh**, it serves that JSON; otherwise it scrapes and updates.

> The table is a **cache**, not a source of truth. Long-term retention is not required. (Later, you can introduce a history table or a KVS layer for faster hits.)

---

## How to connect to MariaDB and verify with SQL

### From the host (Docker published port)

> Compose publishes container port 3306 to **host port 3307**.

```bash
# root
mysql -h 127.0.0.1 -P 3307 -uroot -p

# app user
mysql -h 127.0.0.1 -P 3307 -u "$MARIADB_USER" -p"$MARIADB_PASSWORD" "$MARIADB_DATABASE"
```

> **Always pass `-h 127.0.0.1`** when using the host client. Without `-h`, the client may try a UNIX socket and hit a different local server.

### Quick verification queries

```sql
SHOW DATABASES;
USE jpx;
SHOW TABLES;
DESC scrapings;
SELECT ticker, LENGTH(jsond) AS bytes, updated FROM scrapings ORDER BY updated DESC LIMIT 5;
SELECT jsond FROM scrapings WHERE ticker='8306';
```

If you delete a row to force a refresh:
```sql
DELETE FROM scrapings WHERE ticker='8306';
```

---

## How caching works today (DB-backed)

- The service tries to **serve from DB** if the row is **fresh** (by default ~1 hour freshness window).
- If **stale or missing**, the service **scrapes**, **REPLACE**s the row, and returns the fresh JSON.
- Consecutive requests within the freshness window do **not** hit Selenium again; they read from DB.

### Change the lifetime (freshness window)

Look for the **staleness/TTL constant** in the code (develop branch), typically in the “usecase” or handler layer that decides:
```go
// pseudo: consider DB value fresh if updated within 1 hour
staleWindow := time.Hour
if time.Since(updatedAt) < staleWindow {
    return fromDB
}
```
> Increase/decrease `staleWindow` (e.g., `30*time.Minute`, `2*time.Hour`) to control when the scraper re-runs.

Later (when KVS is added), the DB freshness window can be lengthened (e.g., 30 days) and a **KVS TTL** (e.g., 1–24 hours) can serve the hot path.

---

## End-to-end verification

1) Start the stack:
```bash
docker compose --env-file .env up -d --build
```

2) Call the API twice (second call should be fast and DB-backed):
```bash
time curl -s "http://localhost:8082/scrape?ticker=8306" >/dev/null
time curl -s "http://localhost:8082/scrape?ticker=8306" >/dev/null
```

3) Confirm cache content:
```bash
mysql -h 127.0.0.1 -P 3307 -u "$MARIADB_USER" -p"$MARIADB_PASSWORD" "$MARIADB_DATABASE" \
  -e "SELECT ticker, LENGTH(jsond) bytes, updated FROM scrapings WHERE ticker='8306'"
```

---

## Environment & Compose (compact profile)

Key variables (from `.env`):
```env
# MariaDB
MARIADB_ROOT_PASSWORD=********
MARIADB_DATABASE=jpx
MARIADB_USER=jpx
MARIADB_PASSWORD=********

# App DB connection
DB_HOST=mariadb
DB_PORT=3306
DB_NAME=jpx
DB_USER=jpx
DB_PASSWORD=********
DB_PARAMS=parseTime=true&charset=utf8mb4&loc=Asia%2FTokyo

# Selenium (compact defaults)
SE_NODE_MAX_SESSIONS=1
```

Health checks (Compose):
- MariaDB: `mariadb-admin ping --protocol=TCP -h 127.0.0.1 ...`
- Selenium Hub: `curl -fsS http://localhost:4444/status`
- Scraper waits until MariaDB/Selenium are reachable.

---

## Troubleshooting quicksheet

- **Connected to the wrong DB**: always include `-h 127.0.0.1 -P 3307` on the host, or exec into the container:
  ```bash
  docker exec -it mariadb sh -lc 'mariadb -uroot -p"$MARIADB_ROOT_PASSWORD" -e "SHOW DATABASES"'
  ```
- **Table not found**: the app **creates** `scrapings` at start; check scraper logs:
  ```bash
  docker logs jpx-scraper | grep -i schema
  ```
- **Rebuild didn’t pick code changes**: force a clean rebuild + recreate:
  ```bash
  docker compose build --no-cache scraper
  docker compose up -d --force-recreate scraper
  ```
- **Slow responses**: confirm DB hit rate by watching updated times; reduce `staleWindow` only if necessary.

---

## Project boundaries (what this app does / doesn’t)

**Does:**
- Scrapes Kabutan on demand.
- Caches the JSON into `scrapings` (one row per ticker).
- Serves JSON via REST (`/scrape?ticker=XXXX`).

**Does not (yet):**
- Provide gRPC (will live on a separate port later).
- Use KVS (Valkey) for hot-cache (planned).
- Store long-term history (consider a `quotes` table later if needed).

---

## Roadmap (when ready)

- **KVS(Valkey)** cache-aside: `quote:latest:<ticker>` TTL=1h, DB retention 30d.
- **SWR** (stale-while-revalidate): serve old quickly, refresh async.
- **gRPC** on :9090 (REST remains on :8081).
- **Worker/Scheduler** to decouple scraping from HTTP path.

---

## License
MIT

