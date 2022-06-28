.PHONY: load-checker prepare vendor

load-checker: ## Ensure Souin is running
	cd examples && echo "MIDDLEWARE_RESULT=$$(docker-compose logs souin | grep Souin)"

prepare: vendor ## Prepare souin plugin
	cd examples && cp docker-compose.yml.test docker-compose.yml && docker-compose up -d

vendor: ## Update the go mods for the plugin
	echo "Nothing to do for Souin itself"
