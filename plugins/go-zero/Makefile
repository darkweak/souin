.PHONY: load-checker prepare vendor

load-checker: ## Ensure Souin is running
	cd examples && echo "MIDDLEWARE_RESULT=$$(docker-compose logs go-zero | grep Souin | tail -n 1)"

prepare: vendor ## Prepare go-zero plugin
	cd examples && cp docker-compose.yml.test docker-compose.yml && docker-compose up -d

vendor: ## Update the go mods for the plugin
	go install github.com/zeromicro/go-zero/tools/goctl@latest
	cd examples && goctl api go -api sample.api -dir .
	go mod tidy
	go mod download
