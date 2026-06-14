#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${ENV_FILE:-.env}"
NODES="${NODES:-1}"

compose() {
  docker compose \
    -f docker-compose.yml \
    -f docker-compose.wsl.yml \
    --env-file "$ENV_FILE" \
    "$@"
}

usage() {
  cat <<'EOF'
Usage:
  bash scripts/wsl-compose.sh up
  bash scripts/wsl-compose.sh down
  bash scripts/wsl-compose.sh ps
  bash scripts/wsl-compose.sh logs [service]
  bash scripts/wsl-compose.sh restart
EOF
}

cmd="${1:-}"
shift || true

case "$cmd" in
  up)
    compose up -d --build --scale selenium-node="$NODES" "$@"
    ;;
  down)
    compose down "$@"
    ;;
  ps)
    compose ps "$@"
    ;;
  logs)
    service="${1:-scraper}"
    shift || true
    compose logs -f --tail=100 "$service" "$@"
    ;;
  restart)
    compose up -d --build --force-recreate --scale selenium-node="$NODES" "$@"
    ;;
  *)
    usage
    exit 1
    ;;
esac
