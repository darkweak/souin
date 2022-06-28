.PHONY: load-checker prepare replace vendor

load-checker: ## Ensure Souin is running
	echo "MIDDLEWARE_RESULT=$$(docker-compose logs tyk-gateway | grep Souin)"

prepare: vendor ## Prepare tyk plugin
	cp docker-compose.yml.test docker-compose.yml
	docker-compose down --remove-orphans
	docker-compose up tyk-builder
	mv souin-plugin.so ./samples/middleware/
	docker-compose up -d --remove-orphans --build tyk-pump tyk-gateway

# TODO find another way to do that
replace: ## Replace sources in the vendor folder deeper than the go mod replace
	rm -rf vendor/github.com/darkweak/souin/cache/providers
	scp -r override/providers vendor/github.com/darkweak/souin/cache

vendor: ## Generate vendors for the plugin
	go mod tidy
	go mod download
	go mod vendor
	$(MAKE) replace
