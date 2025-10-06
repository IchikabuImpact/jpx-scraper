#!/bin/bash
set -euo pipefail

if [ ! -f .env ]; then
  echo "Missing .env file. Copy .env.example and configure credentials before deploying." >&2
  exit 1
fi

echo "Building images and starting the stack via Docker Compose..."
docker compose --env-file .env up -d --build

echo "Deployment completed. Use 'docker compose ps' to view container status."
