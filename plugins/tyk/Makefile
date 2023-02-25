.PHONY: load-checker prepare replace vendor

load-checker: ## Ensure Souin is running
	echo "MIDDLEWARE_RESULT=$$(docker compose logs tyk-gateway | grep 'Souin configuration' | tail -n 1)"

prepare: vendor ## Prepare tyk plugin
	cp docker-compose.yml.test docker-compose.yml
	docker compose down --remove-orphans
	docker compose up tyk-builder
	mv souin-plugin.so ./samples/middleware/
	docker compose up -d --remove-orphans --build tyk-pump tyk-gateway

# TODO find another way to do that
replace: ## Replace sources in the vendor folder deeper than the go mod replace
	rm -rf vendor/github.com/darkweak/souin/cache/providers
	scp -r override/providers vendor/github.com/darkweak/souin/cache
	rm -rf vendor/github.com/darkweak/souin/pkg/storage && scp -r override/storage vendor/github.com/darkweak/souin/pkg
	rm -rf vendor/github.com/darkweak/souin/pkg/api && scp -r override/api vendor/github.com/darkweak/souin/pkg
	rm -rf vendor/github.com/darkweak/souin/context && scp -r override/context vendor/github.com/darkweak/souin
	rm -rf vendor/github.com/darkweak/souin/api
	rm -rf vendor/github.com/darkweak/go-esi
	rm -rf vendor/github.com/darkweak/souin/plugins/base.go
	rm -rf vendor/github.com/darkweak/souin/plugins/main.go
	scp override/middleware/middleware.go vendor/github.com/darkweak/souin/pkg/middleware/middleware.go
	scp override/cache/surrogate/providers/cloudflare.go vendor/github.com/darkweak/souin/cache/surrogate/providers/cloudflare.go
	scp override/context/graphql.go vendor/github.com/darkweak/souin/context/graphql.go

vendor: ## Generate vendors for the plugin
	go mod tidy
	go mod download
	go mod vendor
	$(MAKE) replace
