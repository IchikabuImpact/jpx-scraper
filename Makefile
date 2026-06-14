ENV_FILE ?= .env
NODES ?= 1
WSL_COMPOSE = bash scripts/wsl-compose.sh

.PHONY: help wsl-up wsl-down wsl-ps wsl-logs wsl-restart wsl-scrape

help:
	@echo "Targets:"
	@echo "  make wsl-up       Start the WSL-local stack"
	@echo "  make wsl-down     Stop the WSL-local stack"
	@echo "  make wsl-ps       Show WSL-local compose status"
	@echo "  make wsl-logs     Tail scraper logs"
	@echo "  make wsl-restart  Recreate the WSL-local stack"
	@echo "  make wsl-scrape   Call the local scraper API (use TICKER=xxxx)"

wsl-up:
	ENV_FILE=$(ENV_FILE) NODES=$(NODES) $(WSL_COMPOSE) up

wsl-down:
	ENV_FILE=$(ENV_FILE) $(WSL_COMPOSE) down

wsl-ps:
	ENV_FILE=$(ENV_FILE) $(WSL_COMPOSE) ps

wsl-logs:
	ENV_FILE=$(ENV_FILE) $(WSL_COMPOSE) logs scraper

wsl-restart:
	ENV_FILE=$(ENV_FILE) NODES=$(NODES) $(WSL_COMPOSE) restart

wsl-scrape:
	@curl -s "http://127.0.0.1:$${SCRAPER_HOST_PORT:-18082}/scrape?ticker=$${TICKER:-8306}"
