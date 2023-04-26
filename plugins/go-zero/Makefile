.PHONY: load-checker prepare vendor

DC=docker-compose -f docker-compose.yml.test 

load-checker: ## Ensure Souin is running
	cd examples && echo "MIDDLEWARE_RESULT=$$($(DC) logs go-zero | grep 'Souin configuration' | tail -n 1)"

prepare: vendor ## Prepare go-zero plugin
	cd examples && $(DC) up -d --build

vendor: ## Update the go mods for the plugin
	go install github.com/zeromicro/go-zero/tools/goctl@latest
	cd examples && goctl api go -api sample.api -dir .
	go mod tidy
	go mod download
