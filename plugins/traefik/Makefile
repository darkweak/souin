.PHONY: copy-file-to copy-to load-checker prepare replace vendor

SOUIN=vendor/github.com/darkweak/souin
API=$(SOUIN)/api
CACHE=$(SOUIN)/cache

copy-to: ## Copy from to
	rm -rf $(base)/$(target)
	scp -r override/$(target) $(base)

copy-file-to: ## Copy file from to
	rm -rf $(base)/$(target)
	scp -r override/$(target) $(base)/$(target)

load-checker: ## Ensure Souin is running
	echo "MIDDLEWARE_RESULT=$$(docker-compose logs traefik | grep Souin | tail -n 1)"

prepare: vendor ## Prepare traefik plugin
	cp docker-compose.yml.test docker-compose.yml
	docker-compose down --remove-orphans
	docker-compose up -d

# TODO find another way to do that
replace: ## Replace sources in the vendor folder deeper than the go mod replace
	$(MAKE) copy-to base=$(CACHE) target=coalescing
	$(MAKE) copy-to base=$(CACHE) target=providers
	$(MAKE) copy-to base=$(SOUIN) target=context
	$(MAKE) copy-file-to base=$(CACHE) target=surrogate/providers/common.go
	$(MAKE) copy-file-to base=$(CACHE) target=types/layerStorage.go
	$(MAKE) copy-to base=$(CACHE) target=ykeys
	$(MAKE) copy-to base=$(API) target=api/prometheus
	$(MAKE) copy-to base=$(API) target=api/main.go

vendor: ## Generate vendors for the plugin
	go mod tidy
	go mod download
	go mod vendor
	$(MAKE) replace
