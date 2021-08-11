.PHONY: prepare

prepare: ## Prepare tyk plugin
	go mod tidy && \
	go mod download && \
	go mod vendor && \
	$(MAKE) replace

# TODO find another way to do that
replace: ## Replace sources in the vendor folder deeper than the go mod replace
	rm -rf vendor/github.com/darkweak/souin/cache/providers
	scp -r override/providers vendor/github.com/darkweak/souin/cache
