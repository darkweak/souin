.PHONY: load-checker prepare vendor

load-checker: ## Ensure Souin is running
	cd examples && echo "MIDDLEWARE_RESULT=$$(docker-compose -f docker-compose.yml.test logs hertz | grep 'Souin configuration' | tail -n 1)"

prepare: vendor ## Prepare hertz plugin
	cd examples && docker-compose -f docker-compose.yml.test up -d

vendor: ## Update the go mods for the plugin
	go mod tidy
	go mod download
