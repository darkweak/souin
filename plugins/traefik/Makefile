.PHONY: copy-file-to copy-to prepare replace vendor

SOUIN=vendor/github.com/darkweak/souin
CACHE=$(SOUIN)/cache

copy-to: ## Copy from to
	rm -rf $(base)/$(target)
	scp -r override/$(target) $(base)

copy-file-to: ## Copy file from to
	rm -rf $(base)/$(target)
	scp -r override/$(target) $(base)/$(target)

prepare: vendor ## Prepare tyk plugin
	cp docker-compose.yml.test docker-compose.yml
	docker-compose down --remove-orphans
	docker-compose up -d

# TODO find another way to do that
replace: ## Replace sources in the vendor folder deeper than the go mod replace
	$(MAKE) copy-to base=$(CACHE) target=coalescing
	$(MAKE) copy-to base=$(CACHE) target=providers
	$(MAKE) copy-to base=$(CACHE) target=types
	$(MAKE) copy-to base=$(CACHE) target=ykeys
	$(MAKE) copy-file-to base=$(SOUIN) target=rfc/vary.go

vendor: ## Generate vendors for the plugin
	go mod tidy
	go mod download
	go mod vendor
	$(MAKE) replace
