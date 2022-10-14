.PHONY: load-checker prepare vendor

load-checker: ## Ensure Souin is running
	cd examples && echo "MIDDLEWARE_RESULT=$$(docker-compose logs roadrunner | grep 'Souin configuration' | tail -n 1)"

prepare: vendor ## Prepare roadrunner plugin
	cd examples && cp docker-compose.yml.test docker-compose.yml && docker-compose up -d --build

vendor: ## Update the go mods for the plugin
	go mod tidy
	go mod download
