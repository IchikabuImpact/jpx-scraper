# 始値・高値・安値の追加スクレイピング — 要件定義（ドラフト）

作成日: 2026-09-27
作成元: `high-dividend-monthly` プロジェクトからの要望（ローソク足チャート機能のため）

## 背景・目的

`high-dividend-monthly`（月別高配当ダッシュボード）で、日足・週足のローソク足チャート（3ヶ月〜5年）を組んで「ド底」銘柄を視覚的に見つけたい、という要望がある。

現在 `jpx-scraper` の `/scrape?ticker=` が返す `StockData`（`internal/stockdata/stock_data.go`）には `currentPrice` と `previousClose` しかなく、ローソク実体（始値・終値）を描くための**その日の始値が取得できていない**。

比較検討として J-Quants API 経由（`jpx-indicator`, LIGHTプラン）での取得も調査したが、以下の理由で **jpx-scraper 側に項目追加する方が簡単** という結論になった:
- Kabutan の同一ページに始値・高値・安値・終値が既に整形済みテーブルとして存在する（後述）ため追加スクレイピングだけで済む。
- J-Quants LIGHTプランで日足の始値・終値が取得可能かどうか未確認で、確認の手間・プラン変更コストが発生し得る。
- `jpx-scraper` は既に稼働中で他プロジェクトからも使われており、フィールド追加は後方互換（追加のみ）で影響が小さい。

## 確認済み事実（2026-09-27 時点で実ページを確認）

`https://kabutan.jp/stock/?code=8306` の `#kobetsu_left` 内、既存コードが `previousClose` を取っている `<dl><dd>` の**直後**に、始値・高値・安値・終値が入った `<table>` がある:

```html
<div id="kobetsu_left">
<dl>
  <dt>前日終値</dt>
  <dd class="floatr">3,573.0&nbsp;(<time datetime="2026-09-24">09/24</time>)</dd>
</dl>

<h2><time datetime="2026-09-25">09月25日</time></h2>

<table>
  <tbody>
    <tr><th scope='row'>始値</th><td>3,610.0</td><td class="mark">&nbsp;</td><td>(<time datetime="2026-09-25T09:00+09:00">09:00</time>)</td></tr>
    <tr><th scope='row'>高値</th><td>3,719.0</td>...</tr>
    <tr><th scope='row'>安値</th><td>3,608.0</td>...</tr>
    <tr><th scope='row'>終値</th><td>3,714.0</td>...</tr>
  </tbody>
</table>
...
</div><!--kobetsu_left-->
```

- 各行は `th`（ラベル: 始値/高値/安値/終値）+ `td`（値）の並びなので、`th` のテキストでマッチさせれば行の並び順が変わっても頑健に取得できる。
- 既存の `currentPrice`（`.si_i1_2 .kabuka` から取得しているリアルタイム表示）とはDOM上の別要素。今回追加する「終値」は上記テーブルの値（大引け時点の公式終値）で、意味が異なる点に注意。

## 変更内容（提案）

### 1. `StockData` 構造体にフィールド追加（`internal/stockdata/stock_data.go`）

```go
type StockData struct {
    Ticker        string `json:"ticker"`
    CompanyName   string `json:"companyName"`
    CurrentPrice  string `json:"currentPrice"`
    PreviousClose string `json:"previousClose"`
    Open          string `json:"open,omitempty"`   // 追加: 始値
    High          string `json:"high,omitempty"`   // 追加: 高値
    Low           string `json:"low,omitempty"`    // 追加: 安値
    Close         string `json:"close,omitempty"`  // 追加: 終値（公式引け値。currentPriceとは別物）
    DividendYield string `json:"dividendYield"`
    PER           string `json:"per,omitempty"`
    PBR           string `json:"pbr,omitempty"`
    MarketCap     string `json:"marketCap,omitempty"`
    Volume        string `json:"volume,omitempty"`
}
```

### 2. スクレイピングロジック追加

`#kobetsu_left table:nth-of-type(1) tbody tr` を走査し、`th` のテキストが「始値」「高値」「安値」「終値」に一致する行の最初の `td` を取得する（`nth-child` 固定インデックスではなく `th` テキストマッチにすることで、行順の変化に強くする）。

### 3. 後方互換性

- 追加フィールドは全て `omitempty` の追加のみ。既存キー（`ticker`, `currentPrice`, `previousClose` など）は変更しない。
- 呼び出し元は Go/JSの標準デコーダで未知フィールドは無視される実装のため、破壊的変更なし（下記「影響範囲」で確認済み）。

## 影響範囲（呼び出し元）

`jpx-scraper` の `/scrape` を実際に呼んでいるのを確認できたプロジェクト:

1. **`high-dividend-monthly`** (`src/jpxScraper.js`) — 今回の要望元。取得後 `buildSnapshotColumns` で `price_snapshots` に保存する処理があるので、`open_value` / `high_value` / `low_value` 列を追加してマッピングする改修が別途必要。
2. **`stocks2db`** (`internal/fetcher/price_api.go`) — `currentPrice` のみ参照（`priceAPIResponse` 構造体は `ticker`/`currentPrice` のみ）。標準 `json.Decoder` で未知フィールドは無視されるため、フィールド追加による影響なし。

VPS環境では Apache 経由 (`ProxyPass /scrape http://localhost:8082/scrape`) で外部公開されている記録もある（`jitaku-network/docs/home-it-cms-research-notes.md`, `AK1PLUS-WSL-ops.md`）ため、他に把握していない呼び出し元がいないか、実装前に念のため確認したほうがよい。

## 未確定・要確認事項

- 「終値」（大引け公式値）を追加する必要が実際にあるか（`currentPrice` を15:30スクレイピング時点の値として代用できるなら `Close` フィールドは省略しても良い）。
- 上場廃止直後の銘柄・売買停止銘柄など、テーブルの行が欠ける/文言が変わるケースのフォールバック処理。
- 週足への集計は `high-dividend-monthly` 側（またはダッシュボード表示層）で日足を束ねる想定。`jpx-scraper` 側では日足の始値・高値・安値・終値の提供までで完結する。

## 参照

- 要望の背景: `high-dividend-monthly/README.md` の「運用メモ」セクション
- 対になる改修（DB・取り込み側）: `high-dividend-monthly/src/jpxScraper.js`, `src/queue.js`, `sql/schema.sql`（`price_snapshots` へのカラム追加）
