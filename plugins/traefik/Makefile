.PHONY: load-checker prepare vendor

DC=docker compose -f docker-compose.yml.test

load-checker: ## Ensure Souin is running
	echo "MIDDLEWARE_RESULT=$$($(DC) logs traefik | grep 'Souin configuration' | tail -n 1)"

prepare: vendor ## Prepare traefik plugin
	$(DC) down --remove-orphans
	$(DC) up -d

vendor: ## Generate vendors for the plugin
	go mod tidy
	go mod download
	go mod vendor
