.PHONY: load-checker prepare vendor

load-checker: ## Ensure Souin is running
	cd examples && echo "MIDDLEWARE_RESULT=$$(docker-compose -f docker-compose.yml.test logs echo | grep 'Souin configuration' | tail -n 1)"

prepare: vendor ## Prepare echo plugin
	cd examples && docker-compose -f docker-compose.yml.test up -d --build

vendor: ## Update the go mods for the plugin
	go mod tidy
	go mod download
