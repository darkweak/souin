.PHONY: prepare vendor

prepare: vendor ## Prepare tyk plugin
	cd examples && cp docker-compose.yml.test docker-compose.yml && docker-compose up -d

vendor: ## Generate vendors for the plugin
	go mod tidy
	go mod download
