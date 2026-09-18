.PHONY: help local-cache-tests

help:
	@grep -E '(^[0-9a-zA-Z_-]+:.*?##.*$$)|(^##)' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-25s\033[0m %s\n", $$1, $$2}' | sed -e 's/\[32m##/[33m/'

local-cache-tests: ## Run local cache-tests
	if cd .local-cache-tests; then git pull; else git clone https://github.com/http-tests/cache-tests .local-cache-tests; fi
	xcaddy build --with github.com/darkweak/souin/plugins/caddy=./ --with github.com/darkweak/souin=../..
	./caddy run --config ../../docs/cache-tests/cache-tests-caddyfile --adapter caddyfile &
	cp ../../docs/cache-tests/index.mjs .local-cache-tests/results/index.mjs
	cd .local-cache-tests && pnpm install && pnpm run server &
	cd .local-cache-tests && ./test-host.sh 127.0.0.1:4443 > results/mr.json
	killall node caddy