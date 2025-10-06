#!/usr/bin/env bash
set -euo pipefail

# ========= Config =========
ENV_FILE="${ENV_FILE:-.env}"          # 例: ENV_FILE=.env.production
NODES_DEFAULT="${NODES:-4}"           # 例: NODES=2
WAIT_TIMEOUT="${WAIT_TIMEOUT:-120}"   # ヘルス待ち秒数
PRUNE_MODE="${PRUNE_MODE:-deep}"      # deep|safe|none  (既定: deep = ボリューム含め全部prune)
NODES="${1:-$NODES_DEFAULT}"          # 第1引数でノード数指定可

# ========= Pre-check =========
[[ -f "$ENV_FILE" ]] || { echo "Missing $ENV_FILE. Copy .env.example and configure credentials before deploying." >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "docker not found" >&2; exit 1; }
docker compose version >/dev/null 2>&1 || { echo "'docker compose' CLI not found" >&2; exit 1; }

# ========= Prune =========
echo "PRUNE_MODE=${PRUNE_MODE}"
echo "Stopping/removing current compose stack (with orphans)…"
case "$PRUNE_MODE" in
  deep)
    # 全消し（ボリューム含む）。DBもキャッシュ扱いなのでOKとのこと。
    docker compose --env-file "$ENV_FILE" down -v --remove-orphans || true
    docker container prune -f || true
    docker network prune -f || true
    docker image prune -a -f || true
    docker volume prune -f || true
    docker builder prune -a -f || true
    ;;
  safe)
    # ボリュームは残す
    docker compose --env-file "$ENV_FILE" down --remove-orphans || true
    docker container prune -f || true
    docker network prune -f || true
    docker image prune -a -f || true
    docker builder prune -a -f || true
    ;;
  none)
    # 触らない
    echo "Skip pruning (PRUNE_MODE=none)."
    ;;
  *)
    echo "Unknown PRUNE_MODE: $PRUNE_MODE (use deep|safe|none)" >&2
    exit 1
    ;;
esac

# ========= Helpers =========
wait_healthy() {
  local svc="$1"
  local deadline=$((SECONDS + WAIT_TIMEOUT))
  echo "Waiting for service '$svc' to become healthy (timeout: ${WAIT_TIMEOUT}s)…"
  while true; do
    local status
    status="$(docker inspect -f '{{.State.Health.Status}}' "$svc" 2>/dev/null || true)"
    if [[ "$status" == "healthy" ]]; then
      echo "Service '$svc' is healthy."
      return 0
    fi
    if (( SECONDS >= deadline )); then
      echo "Timed out waiting for '$svc' to be healthy." >&2
      docker ps
      return 1
    fi
    sleep 3
  done
}

# ========= Deploy =========
echo "Pulling images (if newer exist)…"
docker compose --env-file "$ENV_FILE" pull

echo "Building & starting with ${NODES} selenium node(s)…"
docker compose --env-file "$ENV_FILE" up -d --build --scale selenium-node="${NODES}"

# ========= Post-check =========
wait_healthy mariadb
wait_healthy selenium-hub || true   # 厳格にするなら || true を外す

echo "Deployment completed."
echo "• Nodes: ${NODES}"
echo "• Prune: ${PRUNE_MODE}"
echo "• Check:  docker compose ps"
echo "• Logs:   docker logs mariadb --tail=50"

