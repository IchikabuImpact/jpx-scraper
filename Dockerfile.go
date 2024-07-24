# ベースイメージを指定
FROM golang:1.22-alpine

# 作業ディレクトリを設定
WORKDIR /app

# Goのモジュールファイルをコピー
COPY go.mod ./
COPY go.sum ./

# 依存関係をダウンロード
RUN go mod download

# ソースコードをコピー
COPY . .

# アプリケーションをビルド
RUN go build -o scraper ./cmd/scraper

# ポートを公開
EXPOSE 8081

# アプリケーションを実行
CMD ["./scraper"]

