# ベースイメージを指定
FROM golang:1.22-alpine

# 必要なパッケージをインストール
RUN apk add --no-cache gcc musl-dev sqlite wget nmap curl

# 作業ディレクトリを設定
WORKDIR /app

# Goのモジュールファイルをコピー
COPY go.mod ./
COPY go.sum ./

# 依存関係をダウンロード
RUN go mod download

# ソースコードをコピー
COPY . .

# CGO_ENABLEDを有効にしてアプリケーションをビルド
ENV CGO_ENABLED=1
RUN go build -o scraper ./cmd/scraper

# ポートを公開
EXPOSE 8081

# アプリケーションを実行
CMD ["./scraper"]

