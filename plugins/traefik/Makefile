.PHONY: copy-file-to copy-to load-checker prepare replace vendor

SOUIN=vendor/github.com/darkweak/souin
PKG=$(SOUIN)/pkg
API=$(SOUIN)/api
PKG_API=$(PKG)/api
CACHE=$(SOUIN)/cache
DC=docker compose -f docker-compose.yml.test

copy-to: ## Copy from to
	rm -rf $(base)/$(target)
	scp -r override/$(target) $(base)

copy-file-to: ## Copy file from to
	rm -rf $(base)/$(target)
	scp -r override/$(target) $(base)/$(target)

load-checker: ## Ensure Souin is running
	echo "MIDDLEWARE_RESULT=$$($(DC) logs traefik | grep 'Souin configuration' | tail -n 1)"

prepare: vendor ## Prepare traefik plugin
	$(DC) down --remove-orphans
	$(DC) up -d

# TODO find another way to do that
replace: ## Replace sources in the vendor folder deeper than the go mod replace
	$(MAKE) copy-to base=$(SOUIN) target=configurationtypes
	$(MAKE) copy-to base=$(SOUIN) target=context
	$(MAKE) copy-file-to base=$(PKG) target=surrogate/providers/common.go
	$(MAKE) copy-to base=$(CACHE) target=ykeys
	$(MAKE) copy-to base=$(API) target=api
	$(MAKE) copy-to base=$(PKG) target=api
	$(MAKE) copy-to base=$(PKG) target=storage
	$(MAKE) copy-file-to base=$(PKG) target=rfc/vary.go
	$(MAKE) copy-file-to base=$(PKG) target=rfc/revalidation.go
	$(MAKE) copy-file-to base=$(PKG) target=middleware/configuration.go
	$(MAKE) copy-file-to base=$(PKG) target=middleware/middleware.go
	$(MAKE) copy-file-to base=$(PKG) target=middleware/writer.go

vendor: ## Generate vendors for the plugin
	go mod tidy
	go mod download
	go mod vendor
	$(MAKE) replace
