#!/bin/bash

# スクリプトがエラーで停止するように設定
set -e

# 古いコンテナを停止して削除
echo "Stopping and removing old containers..."
docker stop $(docker ps -q --filter "name=jpx-scraper") 2>/dev/null || true
docker stop $(docker ps -q --filter "name=selenium-hub") 2>/dev/null || true
docker stop $(docker ps -q --filter "name=selenium-node") 2>/dev/null || true
docker rm $(docker ps -aq --filter "name=jpx-scraper") 2>/dev/null || true
docker rm $(docker ps -aq --filter "name=selenium-hub") 2>/dev/null || true
docker rm $(docker ps -aq --filter "name=selenium-node") 2>/dev/null || true

# 不要なコンテナの削除
docker rm $(docker ps -aq --filter "status=exited") 2>/dev/null || true

# Dockerイメージをビルド
echo "Building Docker images..."
docker build -t jpx-scraper -f Dockerfile.go .
docker build -t selenium-hub-custom -f Dockerfile.selenium .

# ネットワークが存在しない場合に作成
docker network create selenium-network 2>/dev/null || true

# Selenium Hub と Node を起動
echo "Starting Selenium Hub and Node..."
docker run -d --restart unless-stopped -p 4444:4444 --name selenium-hub --network selenium-network selenium-hub-custom
docker run -d --restart unless-stopped --network selenium-network --link selenium-hub:hub -e SE_EVENT_BUS_HOST=selenium-hub -e SE_EVENT_BUS_PUBLISH_PORT=4442 -e SE_EVENT_BUS_SUBSCRIBE_PORT=4443 selenium/node-chrome

# JPX Scraper を起動
echo "Starting JPX Scraper..."
docker run -d --restart unless-stopped -p 8082:8081 --network selenium-network --name jpx-scraper jpx-scraper

echo "Deployment completed successfully."

