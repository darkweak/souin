.PHONY: copy-file-to copy-to load-checker prepare replace vendor

SOUIN=vendor/github.com/darkweak/souin
PKG=$(SOUIN)/pkg
API=$(SOUIN)/api
PKG_API=$(PKG)/api
CACHE=$(SOUIN)/cache
STORAGES=vendor/github.com/darkweak/storages
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
	$(MAKE) copy-to base=$(SOUIN) target=context
	$(MAKE) copy-file-to base=$(PKG) target=surrogate/providers/common.go
	$(MAKE) copy-to base=$(API) target=api
	$(MAKE) copy-to base=$(PKG) target=api
	$(MAKE) copy-file-to base=$(PKG) target=storage/storages.go
	$(MAKE) copy-file-to base=$(PKG) target=middleware/middleware.go
	$(MAKE) copy-file-to base=$(PKG) target=middleware/writer.go
	# Yaegi can't compile protobuf, the storages core encodes its mappings as JSON instead
	rm -rf $(STORAGES)/core
	cp -R override/storages/core $(STORAGES)/core
	# Yaegi can't compile these generic caches, replaced by concrete implementations
	rm -rf vendor/github.com/maypok86/otter vendor/github.com/jellydator/ttlcache
	mkdir -p vendor/github.com/maypok86 vendor/github.com/jellydator
	cp -R override/third_party/maypok86/otter vendor/github.com/maypok86/otter
	cp -R override/third_party/jellydator/ttlcache vendor/github.com/jellydator/ttlcache
	find vendor/github.com/maypok86 vendor/github.com/jellydator -name "*_test.go" -delete
	for p in override/patches/*.patch; do patch -s -p1 -d vendor < $$p || exit 1; done

vendor: ## Generate vendors for the plugin
	go mod tidy
	go mod download
	go mod vendor
	$(MAKE) replace
