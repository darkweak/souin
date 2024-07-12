.PHONY: build-and-run-caddy build-and-run-caddy-json build-and-run-chi build-and-run-dotweb build-and-run-echo build-and-run-fiber \
	build-and-run-gin build-and-run-go-zero build-and-run-goyave build-and-run-skipper build-and-run-souin build-and-run-traefik \
	build-and-run-tyk build-and-run-webgo build-app build-caddy build-dev bump-version coverage create-network down env-dev \
	env-prod gatling generate-plantUML golangci-lint health-check-prod help lint log tests up validate vendor-plugins

DC=docker-compose
DC_BUILD=$(DC) build
DC_EXEC=$(DC) exec
PLUGINS_LIST=beego caddy chi dotweb echo fiber gin goa go-zero goyave hertz kratos roadrunner skipper traefik tyk webgo souin
MOD_PLUGINS_LIST=beego caddy chi dotweb echo fiber gin goa go-zero goyave hertz kratos roadrunner skipper webgo

base-build-and-run-%:
	cd plugins/$* && $(MAKE) prepare

build-and-run-beego: base-build-and-run-beego  ## Run Beego with Souin as plugin

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

build-and-run-hertz: base-build-and-run-hertz  ## Run Hertz with Souin as plugin

build-and-run-kratos: base-build-and-run-kratos  ## Run Kratos with Souin as plugin

build-and-run-roadrunner: base-build-and-run-roadrunner  ## Run Roadrunner with Souin as plugin

build-and-run-skipper: base-build-and-run-skipper  ## Run Skipper with Souin as plugin

build-and-run-souin: base-build-and-run-souin  ## Run Souin as plugin

build-and-run-gin: base-build-and-run-gin  ## Run Gin with Souin as plugin

build-and-run-goa: base-build-and-run-goa  ## Run Goa with Souin as plugin

build-and-run-go-zero: base-build-and-run-go-zero  ## Run Go-zero with Souin as plugin

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
	XCADDY_RACE_DETECTOR=1 XCADDY_DEBUG=1 xcaddy build \
		--with github.com/darkweak/souin/plugins/caddy=./ \
		--with github.com/darkweak/souin=../.. \
		--with github.com/darkweak/storages/badger/caddy \
		--with github.com/darkweak/storages/etcd/caddy \
		--with github.com/darkweak/storages/nuts/caddy \
		--with github.com/darkweak/storages/olric/caddy \
		--with github.com/darkweak/storages/otter/caddy \
		--with github.com/darkweak/storages/redis/caddy

build-caddy-dev: ## Build caddy binary
	cd plugins/caddy && \
	go mod tidy && \
	go mod download && \
	XCADDY_RACE_DETECTOR=1 XCADDY_DEBUG=1 xcaddy build \
		--with github.com/darkweak/souin/plugins/caddy=./ \
		--with github.com/darkweak/souin=../.. \
		--with github.com/darkweak/storages/badger/caddy=../../../storages/badger/caddy \
		--with github.com/darkweak/storages/etcd/caddy=../../../storages/etcd/caddy \
		--with github.com/darkweak/storages/nuts/caddy=../../../storages/nuts/caddy \
		--with github.com/darkweak/storages/olric/caddy=../../../storages/olric/caddy \
		--with github.com/darkweak/storages/otter/caddy=../../../storages/otter/caddy \
		--with github.com/darkweak/storages/redis/caddy=../../../storages/redis/caddy \
		--with github.com/darkweak/storages/badger=../../../storages/badger \
		--with github.com/darkweak/storages/etcd=../../../storages/etcd \
		--with github.com/darkweak/storages/nuts=../../../storages/nuts \
		--with github.com/darkweak/storages/olric=../../../storages/olric \
		--with github.com/darkweak/storages/otter=../../../storages/otter \
		--with github.com/darkweak/storages/redis=../../../storages/redis \
		--with github.com/darkweak/storages/core=../../../storages/core
	cd plugins/caddy && ./caddy run

build-dev: env-dev ## Build containers with dev env vars
	$(DC_BUILD) souin
	$(MAKE) up

bump-version:
	test $(from)
	test $(to)
	sed -i '' 's/version: $(from)/version: $(to)/' README.md
	for plugin in $(PLUGINS_LIST) ; do \
		sed -i '' 's/github.com\/darkweak\/souin $(from)/github.com\/darkweak\/souin $(to)/' plugins/$$plugin/go.mod ; \
	done

bump-plugins-deps: ## Bump plugins dependencies
	for plugin in $(MOD_PLUGINS_LIST) ; do \
		echo "Update $$plugin..." && cd plugins/$$plugin && go get -u ./... && cd - ; \
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

generate-release-plugins: ## Generate plugin workflow
	bash .github/workflows/generate_release.sh

generate-workflow: ## Generate plugin workflow
	bash .github/workflows/workflow_plugins_generator.sh

golangci-lint: ## Run golangci-lint to ensure the code quality
	docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:v1.57.2 golangci-lint run -v --timeout 180s ./...
	for plugin in $(PLUGINS_LIST) ; do \
		echo "Starting lint $$plugin \n" && docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:v1.57.2 golangci-lint run -v --skip-dirs=override --timeout 240s ./plugins/$$plugin; \
	done
	cd plugins/caddy && go mod tidy && go mod download

health-check-prod: build-app ## Production container health check
	$(DC_EXEC) souin ls

help:
	@grep -E '(^[0-9a-zA-Z_-]+:.*?##.*$$)|(^##)' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-25s\033[0m %s\n", $$1, $$2}' | sed -e 's/\[32m##/[33m/'

lint: ## Run lint
	$(DC_EXEC) souin /app/bin/golint ./...

log: ## Show souin logs
	$(DC) logs -f souin

sync-goreleaser-plugins: ## Synchronize plugins goreleaser
	for plugin in $(PLUGINS_LIST) ; do \
		cp .goreleaser.yml plugins/$$plugin; \
	done

tests: ## Run tests
	$(DC_EXEC) souin go test -v ./...

up: ## Up containers
	$(DC) up -d --remove-orphans

validate: lint tests down health-check-prod ## Run lint, tests and ensure prod can build

vendor-plugins: ## Generate and prepare vendors for each plugin
	go mod tidy && go mod download
	for plugin in $(PLUGINS_LIST) ; do \
		cd plugins/$$plugin && ($(MAKE) vendor || true) && cd -; \
	done
	cd plugins/caddy && go mod tidy && go mod download
