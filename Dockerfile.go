FROM golang:1.25.0-alpine
RUN apk add --no-cache gcc musl-dev sqlite wget nmap curl
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1
RUN go build -o scraper ./cmd/scraper
EXPOSE 8081
CMD ["./scraper"]

