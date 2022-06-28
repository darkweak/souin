.PHONY: load-checker prepare vendor

load-checker: ## Ensure Souin is running
	cd examples && echo "MIDDLEWARE_RESULT=$$(docker-compose logs webgo | grep Souin)"

prepare: vendor ## Prepare webgo plugin
	cd examples && cp docker-compose.yml.test docker-compose.yml && docker-compose up -d

vendor: ## Update the go mods for the plugin
	go mod tidy
	go mod download
