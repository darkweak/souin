module github.com/darkweak/souin/plugins/caddy

go 1.15

require (
	github.com/caddyserver/caddy/v2 v2.4.0-beta.2
	github.com/darkweak/souin v1.5.2
	go.uber.org/zap v1.19.0
)

replace github.com/darkweak/souin v1.5.2 => ../..
