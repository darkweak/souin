+++
weight = 601
title = "Caching on wordpress using Caddy"
icon = "deployed_code"
description = "🚀 Blazing fast Wordpress + Caddy"
tags = ["Beginners"]
+++

## What is Wordpress
Wordpress is a web content management system. It was originally created as a tool to publish blogs but has evolved to support publishing other web content, including more traditional websites, mailing lists and Internet forum, media galleries, membership sites, learning management systems and online stores.

## Minimalistic setup

### Caddyfile
The following Caddyfile will enable Souin as cache system in caddy. We set dynamically the server name using the environment variable.
```
{
    default_sni {$SERVER_NAME}
    order cache before rewrite
    cache {
        api {
            souin
        }
    }
}

{$SERVER_NAME} {
    cache
    root * /var/www/html
    encode zstd gzip

    php_fastcgi wordpress:9000
    file_server

    log {
        output file /var/log/caddy.log
    }

    header / {
        X-Frame-Options "SAMEORIGIN"
        X-Content-Type-Options "nosniff"
    }

}
```

### Docker setup
You just need to setup a `compose.yaml` file that will contain the 3 needed services, `caddy`, `wordpress`, `mysql`:
```yaml
version: '3.8'

services:
  caddy:
    build:
      context: .
    environment:
      SERVER_NAME: localhost
    volumes:
      - wordpress:/var/www/html
      - ./Caddyfile:/etc/caddy/Caddyfile
    ports:
      - 80:80
      - 443:443
      - 443:443/udp

  wordpress:
    image: wordpress:fpm
    restart: always
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: demo
      WORDPRESS_DB_PASSWORD: demo
      WORDPRESS_DB_NAME: demo
    volumes:
      - wordpress:/var/www/html

  db:
    image: mysql:8.0
    restart: always
    environment:
      MYSQL_DATABASE: demo
      MYSQL_USER: demo
      MYSQL_PASSWORD: demo
      MYSQL_RANDOM_ROOT_PASSWORD: '1'
    volumes:
      - db:/var/lib/mysql

volumes:
  wordpress:
  db:
```

As we defined the `caddy` service with a custom image, let's define the `Dockerfile` to build the `caddy` instance with `Souin` HTTP cache:
```Dockerfile
FROM caddy:builder-alpine AS builder
RUN xcaddy build --with github.com/darkweak/souin/plugins/caddy

FROM caddy
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
```

And voilà, you're now ready to run:
```
docker compose up -d
```
