+++
weight = 602
title = "HTTP caching with API Platform"
icon = "deployed_code"
description = "🚀🕷️ Blazing fast API Platform"
tags = ["Beginners"]
+++

## What is API Platform
API Platform is a next-generation web framework designed to easily create API-first projects without compromising extensibility and flexibility.

## Minimalistic setup

### Caddyfile
By default API Platform uses FrankenPHP (that is built on top of the Caddy webserver) as reverse-proxy. So we can edit the provided Caddyfile `api/frankenphp/Caddyfile` and configure our caddy instance with these minimal changes:
```diff
{
	{$CADDY_GLOBAL_OPTIONS}

	frankenphp {
		{$FRANKENPHP_CONFIG}
	}
+
+	cache {
+		api {
+			souin
+		}
+		cdn {
+			strategy hard
+		}
+		key {
+			headers Authorization
+		}
+		otter
+	}
}
{$CADDY_EXTRA_CONFIG}

{$SERVER_NAME:localhost} {
	log {
		# Redact the authorization query parameter that can be set by Mercure
		format filter {
			request>uri query {
				replace authorization REDACTED
			}
		}
	}

+	cache
+
	root * /app/public
	encode zstd br gzip

	mercure {
		# Transport to use (default to Bolt)
		transport_url {$MERCURE_TRANSPORT_URL:bolt:///data/mercure.db}
		# Publisher JWT key
		publisher_jwt {env.MERCURE_PUBLISHER_JWT_KEY} {env.MERCURE_PUBLISHER_JWT_ALG}
		# Subscriber JWT key
		subscriber_jwt {env.MERCURE_SUBSCRIBER_JWT_KEY} {env.MERCURE_SUBSCRIBER_JWT_ALG}
		# Allow anonymous subscribers (double-check that it's what you want)
		anonymous
		# Enable the subscription API (double-check that it's what you want)
		subscriptions
		# Extra directives
		{$MERCURE_EXTRA_DIRECTIVES}
	}

	vulcain

	# Add links to the API docs and to the Mercure Hub if not set explicitly (e.g. the PWA)
	header ?Link `</docs.jsonld>; rel="http://www.w3.org/ns/hydra/core#apiDocumentation", </.well-known/mercure>; rel="mercure"`
	# Disable Topics tracking if not enabled explicitly: https://github.com/jkarlin/topics
	header ?Permissions-Policy "browsing-topics=()"

	# Matches requests for HTML documents, for static files and for Next.js files,
	# except for known API paths and paths with extensions handled by API Platform
	@pwa expression `(
			header({'Accept': '*text/html*'})
			&& !path(
				'/docs*', '/graphql*', '/bundles*', '/contexts*', '/_profiler*', '/_wdt*',
				'*.json*', '*.html', '*.csv', '*.yml', '*.yaml', '*.xml'
			)
		)
		|| path('/favicon.ico', '/manifest.json', '/robots.txt', '/sitemap*', '/_next*', '/__next*')
		|| query({'_rsc': '*'})`

	# Comment the following line if you don't want Next.js to catch requests for HTML documents.
	# In this case, they will be handled by the PHP app.
	reverse_proxy @pwa http://{$PWA_UPSTREAM}

	php_server
}
```

### Dockerfile
You now have to update the Dockerfile `api/Dockerfile` to build the FrankenPHP/Caddy instance with Souin (or the cache-handler):
```diff
#syntax=docker/dockerfile:1

# Adapted from https://github.com/dunglas/symfony-docker


# Versions
- FROM dunglas/frankenphp:1-php8.3 AS frankenphp_upstream
+ FROM dunglas/frankenphp:latest-builder AS builder
+ COPY --from=caddy:builder /usr/bin/xcaddy /usr/bin/xcaddy
+ 
+ ENV CGO_ENABLED=1 XCADDY_SETCAP=1 XCADDY_GO_BUILD_FLAGS="-ldflags \"-w -s -extldflags '-Wl,-z,stack-size=0x80000'\""
+ RUN xcaddy build \
+     --output /usr/local/bin/frankenphp \
+     --with github.com/dunglas/frankenphp \
+     --with github.com/dunglas/frankenphp/caddy \
+     --with github.com/dunglas/mercure/caddy \
+     --with github.com/dunglas/vulcain/caddy \
+     --with github.com/dunglas/caddy-cbrotli \
+     # Use this one in production
+     # --with github.com/caddyserver/cache-handler \
+     --with github.com/darkweak/souin/plugins/caddy \
+     --with github.com/darkweak/storages/otter/caddy
+ 
+ FROM dunglas/frankenphp:latest AS frankenphp_upstream
+ COPY --from=builder --link /usr/local/bin/frankenphp /usr/local/bin/frankenphp


# The different stages of this Dockerfile are meant to be built into separate images
# https://docs.docker.com/develop/develop-images/multistage-build/#stop-at-a-specific-build-stage
# https://docs.docker.com/compose/compose-file/#target


# Base FrankenPHP image
FROM frankenphp_upstream AS frankenphp_base

WORKDIR /app

# persistent / runtime deps
# hadolint ignore=DL3008
RUN apt-get update && apt-get install --no-install-recommends -y \
	acl \
	file \
	gettext \
	git \
	&& rm -rf /var/lib/apt/lists/*

# https://getcomposer.org/doc/03-cli.md#composer-allow-superuser
ENV COMPOSER_ALLOW_SUPERUSER=1

RUN set -eux; \
	install-php-extensions \
		@composer \
		apcu \
		intl \
		opcache \
		zip \
	;

###> recipes ###
###> doctrine/doctrine-bundle ###
RUN set -eux; \
	install-php-extensions pdo_pgsql
###< doctrine/doctrine-bundle ###
###< recipes ###

COPY --link frankenphp/conf.d/app.ini $PHP_INI_DIR/conf.d/
COPY --link --chmod=755 frankenphp/docker-entrypoint.sh /usr/local/bin/docker-entrypoint
COPY --link frankenphp/Caddyfile /etc/caddy/Caddyfile

ENTRYPOINT ["docker-entrypoint"]

HEALTHCHECK --start-period=60s CMD curl -f http://localhost:2019/metrics || exit 1
CMD [ "frankenphp", "run", "--config", "/etc/caddy/Caddyfile" ]

# Dev FrankenPHP image
FROM frankenphp_base AS frankenphp_dev

ENV APP_ENV=dev XDEBUG_MODE=off
VOLUME /app/var/

RUN mv "$PHP_INI_DIR/php.ini-development" "$PHP_INI_DIR/php.ini"

RUN set -eux; \
	install-php-extensions \
		xdebug \
	;

COPY --link frankenphp/conf.d/app.dev.ini $PHP_INI_DIR/conf.d/

CMD [ "frankenphp", "run", "--config", "/etc/caddy/Caddyfile", "--watch" ]

# Prod FrankenPHP image
FROM frankenphp_base AS frankenphp_prod

ENV APP_ENV=prod
ENV FRANKENPHP_CONFIG="import worker.Caddyfile"

RUN mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini"

COPY --link frankenphp/conf.d/app.prod.ini $PHP_INI_DIR/conf.d/
COPY --link frankenphp/worker.Caddyfile /etc/caddy/worker.Caddyfile

# prevent the reinstallation of vendors at every changes in the source code
COPY --link composer.* symfony.* ./
RUN set -eux; \
	composer install --no-cache --prefer-dist --no-dev --no-autoloader --no-scripts --no-progress

# copy sources
COPY --link . ./
RUN rm -Rf frankenphp/

RUN set -eux; \
	mkdir -p var/cache var/log; \
	composer dump-autoload --classmap-authoritative --no-dev; \
	composer dump-env prod; \
	composer run-script --no-dev post-install-cmd; \
	chmod +x bin/console; sync;

```

And voilà, your API Platform project has now an HTTP cache in front of your application. But you would probably enable the automatic invalidation to be sure your responses are always up to date, especially to refresh the list of your entity when you create a new item or update one that is in this list.

To do that, you have to update the `api/config/packages/api_platform.yml` file to enable the HTP cache invalidation:
```diff
api_platform:
    title: Hello API Platform
    version: 1.0.0
    # Mercure integration, remove if unwanted
    mercure:
        include_type: true
    formats:
        jsonld: ['application/ld+json']
    docs_formats:
        jsonld: ['application/ld+json']
        jsonopenapi: ['application/vnd.openapi+json']
        html: ['text/html']
    # Good defaults for REST APIs
    defaults:
        stateless: true
        cache_headers:
            vary: ['Content-Type', 'Authorization', 'Origin']
        extra_properties:
            standard_put: true
            rfc_7807_compliant_errors: true
    # change this to true if you use controllers
    use_symfony_listeners: false

+   http_cache:
+       invalidation:
+           urls: [ 'http://php:2019/souin-api/souin' ]
+           purger: api_platform.http_cache.purger.souin
```

You're now ready to handle tons of requests that will be served by the HTTP cache.
