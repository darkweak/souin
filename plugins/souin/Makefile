.PHONY: load-checker prepare vendor

DC=docker compose -f docker-compose.yml.test 

load-checker: ## Ensure Souin is running
	cd examples && echo "MIDDLEWARE_RESULT=$$($(DC) logs souin | grep 'Souin configuration' | tail -n 1)"

prepare: vendor ## Prepare souin plugin
	cd examples && $(DC) up -d --build

vendor: ## Update the go mods for the plugin
	echo "Nothing to do for Souin itself"
