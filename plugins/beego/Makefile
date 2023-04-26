.PHONY: load-checker prepare vendor

DC=docker-compose -f docker-compose.yml.test 

load-checker: ## Ensure Souin is running
	cd examples && echo "MIDDLEWARE_RESULT=$$($(DC) logs beego | grep 'Souin configuration' | tail -n 1)"

prepare: vendor ## Prepare beego plugin
	cd examples && $(DC) up -d --build

vendor: ## Update the go mods for the plugin
	go mod tidy
	go mod download
