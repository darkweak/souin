.PHONY: build-app build-dev coverage create-network down env-dev env-prod gatling generate-plantUML help lint log tests up validate

DC=docker-compose
DC_BUILD=$(DC) build
DC_EXEC=$(DC) exec

build-app: env-prod ## Build containers with prod env vars
	$(DC_BUILD) olric souin
	$(MAKE) up

build-caddy: ## Build caddy binary
	cd plugins/caddy && xcaddy build --with github.com/darkweak/souin/plugins/caddy=./ --with github.com/darkweak/souin@latest=../..

build-and-run-caddy: ## Run caddy binary with the Caddyfile configuration
	$(MAKE) build-caddy
	cd plugins/caddy && ./caddy run

build-and-run-caddy-json:  ## Run caddy binary with the json configuration
	$(MAKE) build-caddy
	cd plugins/caddy && ./caddy run --config ./configuration.json

build-dev: env-dev ## Build containers with dev env vars
	$(DC_BUILD) olric souin
	$(MAKE) up

health-check-prod: build-app ## Production container health check
	$(DC_EXEC) souin ls

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
