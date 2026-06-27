# JPX Scraper — Compact Architecture & Ops Guide (develop)

JPX Scraper is a Go API that scrapes stock data (Kabutan), **caches the JSON in MySQL**, and serves it via HTTP:
```
GET /scrape?ticker=8306
```
## まず直感的に: これで何ができる？

銘柄コード（ticker）を1つ渡すだけで、株価関連の主要指標を **JSONでまとめて取得** できます。

```bash
# VPS 上で 8085 にポート転送している場合の例
curl -i "http://127.0.0.1:8085/scrape?ticker=5020"
```

```json
{
  "ticker":"5020",
  "companyName":"5020　ＥＮＥＯＳホールディングス",
  "currentPrice":"1,488.0円",
  "previousClose":"1,447.5 (02/26)",
  "dividendYield":"1.75％",
  "per":"18.6倍",
  "pbr":"1.3倍",
  "marketCap":"4兆32億円",
  "volume":"11,494,600"
}
```

つまり、フロントエンドや他システムからは「`/scrape?ticker=xxxx` を叩けば、画面表示や分析に使いやすい最新データが返るAPI」として利用できます。

> **⚠️ 利用上の注意 — リクエスト間隔について**
>
> このAPIは内部でKabutanをスクレイピングしています。スクレイピングが発生するのは **キャッシュが古い（または存在しない）場合のみ** ですが、短時間に大量の異なるティッカーをリクエストするとKabutanのサーバーに負荷をかけます。
>
> - **目安: 1〜3リクエスト / 秒以内** に抑えてください
> - 連続して叩く場合は `time.Sleep` や `time.Tick` などで間隔を設けてください

This README focuses on **what gets created in MySQL**, **how to verify with SQL**, and **which knobs control cache lifetime**. It also documents the **compact Docker** profile you’re currently running.

## Which Port Should I Call?

環境ごとに叩くポートが違います。`/scrape` のパス自体は同じで、違うのはホスト側の公開ポートだけです。

| Environment | Start method | Host port | Example |
| --- | --- | --- | --- |
| VPS / port-forwarded host | VPS 側の転送設定を利用 | `8085` | `curl -i "http://127.0.0.1:8085/scrape?ticker=5020"` |
| Local Docker (base compose only) | `docker compose up -d --build` | `8082` | `curl -i "http://127.0.0.1:8082/scrape?ticker=5020"` |
| Local WSL override | `make wsl-up` / `bash scripts/wsl-compose.sh up` | `18082` | `curl -i "http://127.0.0.1:18082/scrape?ticker=5020"` |

- `docker-compose.yml` 単体では `8082:8081`
- `docker-compose.wsl.yml` を重ねると `18082:8081`
- `8085` はアプリのデフォルト公開ポートではなく、VPS 側で別途ポート転送しているときの入口

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
          Scraper (Go) ── DB cache (MySQL 8.x)
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

## How to connect to MySQL and verify with SQL

### From the host (Docker published port)

> Compose publishes container port 3306 to **host port 3307**.

```bash
# root
mysql -h 127.0.0.1 -P 3307 -uroot -p

# app user
mysql -h 127.0.0.1 -P 3307 -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE"
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

### WSL local override

WSL Ubuntu で他プロジェクトから API として再利用するローカル運用では、VPS 用のベース構成はそのままにして `docker-compose.wsl.yml` を重ねます。

短いコマンドで扱うなら、次を使えます。

```bash
make wsl-up
make wsl-ps
make wsl-logs
make wsl-scrape TICKER=8306
make wsl-restart
make wsl-down
```

既定では API は `http://127.0.0.1:18082` で待ち受けます。

または:

```bash
bash scripts/wsl-compose.sh up
bash scripts/wsl-compose.sh ps
bash scripts/wsl-compose.sh logs scraper
bash scripts/wsl-compose.sh down
```

内部的には次の compose 構成を呼びます。

```bash
docker compose \
  -f docker-compose.yml \
  -f docker-compose.wsl.yml \
  --env-file .env \
  up -d --build
```

この override は以下だけをローカル向けに変えます。
- ホスト公開ポートを `13306` / `14444` / `18082` に上げる
- ベースの `docker-compose.yml` はそのまま使い、VPS とローカルで DB 実体を分けない

ローカル `.env` には少なくとも次を入れてください。

```env
MYSQL_ROOT_PASSWORD=********
MYSQL_DATABASE=jpx
MYSQL_USER=jpx
MYSQL_PASSWORD=********
MYSQL_HOST_PORT=13306
SELENIUM_HOST_PORT=14444
SCRAPER_HOST_PORT=18082
```

2) Call the API twice (second call should be fast and DB-backed):
```bash
time curl -s "http://localhost:8082/scrape?ticker=8306" >/dev/null
time curl -s "http://localhost:8082/scrape?ticker=8306" >/dev/null
```

3) Confirm cache content:
```bash
mysql -h 127.0.0.1 -P 3307 -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" \
  -e "SELECT ticker, LENGTH(jsond) bytes, updated FROM scrapings WHERE ticker='8306'"
```

---

## Selenium Grid (single-node) ops checklist

1) Start (single node is the default in this compose file):
```bash
docker compose up -d
```

2) Verify exactly one node container is running and hub is healthy:
```bash
docker ps --filter "name=selenium-node"
docker inspect --format='{{.State.Health.Status}}' jpx-scraper-selenium-hub-1
```

3) Smoke test: confirm the scraper can reach Selenium via the hub:
```bash
curl -s "http://localhost:8082/scrape?ticker=8306" >/dev/null
docker logs jpx-scraper | tail -n 50
```

4) Resource check (low-memory VPS):
```bash
free -hm
docker stats --no-stream
```

---

## Environment & Compose (compact profile)

Key variables (from `.env`):
```env
# MySQL
MYSQL_ROOT_PASSWORD=********
MYSQL_DATABASE=jpx
MYSQL_USER=jpx
MYSQL_PASSWORD=********

# App DB connection
DB_HOST=mysql
DB_PORT=3306
DB_NAME=jpx
DB_USER=jpx
DB_PASSWORD=********
DB_PARAMS=parseTime=true&charset=utf8mb4&loc=Asia%2FTokyo

# Selenium (compact defaults)
SE_NODE_MAX_SESSIONS=1
```

Health checks (Compose):
- MySQL: `mysqladmin ping --protocol=TCP -h 127.0.0.1 ...`
- Selenium Hub: `curl -fsS http://localhost:4444/status`
- Scraper waits until MySQL/Selenium are reachable.

---

## Troubleshooting quicksheet

- **Connected to the wrong DB**: always include `-h 127.0.0.1 -P 3307` on the host, or exec into the container:
  ```bash
  docker compose exec mysql sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "SHOW DATABASES"'
  ```

## VPS rebuild after switching to MySQL

VPS 側では `.env` を MySQL 用に合わせたうえで、キャッシュ DB を作り直す前提なら次で再構築できます。

```env
MYSQL_ROOT_PASSWORD=********
MYSQL_DATABASE=jpx
MYSQL_USER=jpx
MYSQL_PASSWORD=********

DB_HOST=mysql
DB_PORT=3306
DB_NAME=jpx
DB_USER=jpx
DB_PASSWORD=********
```

```bash
PRUNE_MODE=deep ./deploy-jpx-scraper.sh
```

ボリュームも含めて落としてよいなら、次でも同じです。

```bash
docker compose --env-file .env down -v --remove-orphans
docker compose --env-file .env up -d --build
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
