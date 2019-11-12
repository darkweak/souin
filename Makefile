.PHONY: build-app build-dev down env-dev env-prod help up

DC=docker-compose
DC_BUILD=$(DC) build
DC_EXEC=$(DC) exec

build-app: env-prod ## Build containers with prod env vars
	$(DC_BUILD) app
	$(MAKE) up

build-dev: env-dev ## Build containers with dev env vars
	$(DC_BUILD) app
	$(MAKE) up

down: ## Down containers
	$(DC) down --remove-orphans

env-dev: ## Up container with dev env vars
	cp Dockerfile-dev Dockerfile
	cp docker-compose.yml.dev docker-compose.yml
	cp .env.dev .env

env-prod: ## Up container with prod env vars
	cp Dockerfile-prod Dockerfile
	cp docker-compose.yml.prod docker-compose.yml
	cp .env.prod .env

help:
	@grep -E '(^[0-9a-zA-Z_-]+:.*?##.*$$)|(^##)' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-25s\033[0m %s\n", $$1, $$2}' | sed -e 's/\[32m##/[33m/'

up: ## Up containers
	$(DC) up -d --remove-orphans
