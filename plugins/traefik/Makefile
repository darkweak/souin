.PHONY: prepare

SOUIN=vendor/github.com/darkweak/plugins/traefik
CACHE=$(SOUIN)/cache

prepare: ## Prepare tyk plugin
	go mod tidy && \
	go mod download && \
	go mod vendor && \
	$(MAKE) replace && \
	docker-compose up -d whoami
	clear
	docker-compose up traefik

copy-to: ## Copy from to
	rm -rf $(base)/$(target)
	scp -r override/$(target) $(base)

copy-file-to: ## Copy file from to
	rm -rf $(base)/$(target)
	scp -r override/$(target) $(base)/$(target)

# TODO find another way to do that
replace: ## Replace sources in the vendor folder deeper than the go mod replace
	$(MAKE) copy-to base=$(CACHE) target=coalescing
	$(MAKE) copy-to base=$(CACHE) target=providers
	$(MAKE) copy-to base=$(CACHE) target=types
	$(MAKE) copy-to base=$(CACHE) target=ykeys
	$(MAKE) copy-file-to base=$(SOUIN) target=rfc/vary.go
