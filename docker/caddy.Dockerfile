FROM caddy:2.7-builder-alpine AS app_caddy_builder

COPY . /usr/local/go/src/souin
WORKDIR /usr/local/go/src/souin/plugins/caddy

RUN xcaddy build v2.7.4 --with github.com/darkweak/souin/plugins/caddy=./ --with github.com/darkweak/souin=../..
RUN mv ./caddy /usr/bin/caddy

FROM caddy:2-alpine AS app_caddy
WORKDIR /srv/app

COPY --from=app_caddy_builder --link /usr/bin/caddy /usr/bin/caddy
COPY ./plugins/caddy/Caddyfile /etc/caddy/Caddyfile