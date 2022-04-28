.PHONY: prepare vendor

prepare: vendor ## Prepare go-zero plugin
	cd examples && cp docker-compose.yml.test docker-compose.yml && docker-compose up -d

vendor: ## Update the go mods for the plugin
	cd examples && goctl api go -api sample.api -dir .
	go mod tidy
	go mod download
