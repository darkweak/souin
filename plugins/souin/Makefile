.PHONY: prepare vendor

prepare: vendor ## Prepare souin plugin
	cd examples && cp docker-compose.yml.test docker-compose.yml && docker-compose up -d

vendor: ## Update the go mods for the plugin
	echo "Nothing to do for Souin itself"
