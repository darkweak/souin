.PHONY: build-and-run-caddy build-and-run-caddy-json build-and-run-chi build-and-run-dotweb build-and-run-echo build-and-run-fiber \
	build-and-run-gin build-and-run-go-zero build-and-run-goyave build-and-run-skipper build-and-run-souin build-and-run-traefik \
	build-and-run-tyk build-and-run-webgo build-app build-caddy build-dev bump-version coverage create-network down env-dev \
	env-prod gatling generate-plantUML golangci-lint health-check-prod help lint log tests up validate vendor-plugins

DC=docker-compose
DC_BUILD=$(DC) build
DC_EXEC=$(DC) exec
PLUGINS_LIST=caddy chi dotweb echo fiber skipper gin go-zero goyave traefik tyk webgo

base-build-and-run-%:
	cd plugins/$* && $(MAKE) prepare

build-and-run-caddy: ## Run caddy binary with the Caddyfile configuration
	$(MAKE) build-caddy
	cd plugins/caddy && ./caddy run

build-and-run-caddy-json:  ## Run caddy binary with the json configuration
	$(MAKE) build-caddy
	cd plugins/caddy && ./caddy run --config ./configuration.json

build-and-run-chi: base-build-and-run-chi  ## Run Chi with Souin as plugin

build-and-run-dotweb: base-build-and-run-dotweb  ## Run Dotweb with Souin as plugin

build-and-run-echo: base-build-and-run-echo  ## Run Echo with Souin as plugin

build-and-run-fiber: base-build-and-run-fiber  ## Run Fiber with Souin as plugin

build-and-run-skipper: base-build-and-run-skipper  ## Run Skipper with Souin as plugin

build-and-run-souin: base-build-and-run-souin  ## Run Souin as plugin

build-and-run-gin: base-build-and-run-gin  ## Run Gin with Souin as plugin

build-and-run-go-zero: base-build-and-run-go-zero  ## Run Gin with Souin as plugin

build-and-run-goyave: base-build-and-run-goyave  ## Run Goyave with Souin as plugin

build-and-run-traefik: base-build-and-run-traefik  ## Run træfik with Souin as plugin

build-and-run-tyk: base-build-and-run-tyk  ## Run tyk with Souin as middleware

build-and-run-webgo: base-build-and-run-webgo  ## Run Webgo with Souin as plugin

build-app: env-prod ## Build containers with prod env vars
	$(DC_BUILD) souin
	$(MAKE) up

build-caddy: ## Build caddy binary
	cd plugins/caddy && \
	go mod tidy && \
	go mod download && \
	xcaddy build --with github.com/darkweak/souin/plugins/caddy=./ --with github.com/darkweak/souin=../..

build-dev: env-dev ## Build containers with dev env vars
	$(DC_BUILD) souin
	$(MAKE) up

bump-version:
	sed -i '' 's/version: $(from)/version: $(to)/' README.md
	for plugin in $(PLUGINS_LIST) ; do \
        sed -i '' 's/github.com\/darkweak\/souin $(from)/github.com\/darkweak\/souin $(to)/' plugins/$$plugin/go.mod ; \
    done

coverage: ## Show code coverage
	$(DC_EXEC) souin go test ./... -coverprofile cover.out
	$(DC_EXEC) souin go tool cover -func cover.out

create-network: ## Create network
	docker network create your_network

down: ## Down containers
	$(DC) down --remove-orphans

env-dev: ## Up container with dev env vars
	cp Dockerfile-dev Dockerfile
	cp docker-compose.yml.dev docker-compose.yml

env-prod: ## Up container with prod env vars
	cp Dockerfile-prod Dockerfile
	cp docker-compose.yml.prod docker-compose.yml

gatling: ## Launch gatling scenarios
	cd ./gatling && $(DC) up

generate-plantUML: ## Generate plantUML diagrams
	cd ./docs/plantUML && sh generate.sh && cd ../..

golangci-lint: ## Run golangci-lint to ensure the code quality
	docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.42.0 golangci-lint run -v

health-check-prod: build-app ## Production container health check
	$(DC_EXEC) souin ls

help:
	@grep -E '(^[0-9a-zA-Z_-]+:.*?##.*$$)|(^##)' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-25s\033[0m %s\n", $$1, $$2}' | sed -e 's/\[32m##/[33m/'

lint: ## Run lint
	$(DC_EXEC) souin /app/bin/golint ./...

log: ## Show souin logs
	$(DC) logs -f souin

tests: ## Run tests
	$(DC_EXEC) souin go test -v ./...

up: ## Up containers
	$(DC) up -d --remove-orphans

validate: lint tests down health-check-prod ## Run lint, tests and ensure prod can build

vendor-plugins: ## Generate and prepare vendors for each plugin
	for plugin in $(PLUGINS_LIST) ; do \
        cd plugins/$$plugin && ($(MAKE) vendor || true) && cd -; \
    done
	cd plugins/caddy && go mod tidy && go mod download
