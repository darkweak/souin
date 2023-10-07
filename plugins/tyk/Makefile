.PHONY: load-checker prepare replace vendor

DC=docker-compose -f docker-compose.yml.test 

load-checker: ## Ensure Souin is running
	echo "MIDDLEWARE_RESULT=$$($(DC) logs tyk-gateway | grep 'Souin configuration' | tail -n 1)"

prepare: vendor ## Prepare tyk plugin
	$(DC) down --remove-orphans
	$(DC) up tyk-builder
	cp souin-plugin_v5.0.0_linux_amd64.so ./samples/middleware/souin-plugin.so
	mv souin-plugin_v5.0.0_linux_amd64.so ./samples/middleware/
	$(DC) up -d --remove-orphans --build tyk-pump tyk-gateway

# TODO find another way to do that
replace: ## Replace sources in the vendor folder deeper than the go mod replace
	rm -rf vendor/github.com/darkweak/souin/pkg/storage && scp -r override/storage vendor/github.com/darkweak/souin/pkg
	rm -rf vendor/github.com/darkweak/souin/pkg/api && scp -r override/api vendor/github.com/darkweak/souin/pkg
	rm -rf vendor/github.com/darkweak/go-esi
	scp override/middleware/middleware.go vendor/github.com/darkweak/souin/pkg/middleware/middleware.go
	scp override/middleware/writer.go vendor/github.com/darkweak/souin/pkg/middleware/writer.go

vendor: ## Generate vendors for the plugin
	go mod tidy
	go mod download
	go mod vendor
	$(MAKE) replace
