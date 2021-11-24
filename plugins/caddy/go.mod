module github.com/darkweak/souin/plugins/caddy

go 1.16

require (
	github.com/caddyserver/caddy/v2 v2.4.5
	github.com/darkweak/souin v1.5.11-beta1
	go.uber.org/zap v1.19.0
)

replace github.com/darkweak/souin v1.5.11-beta1 => ../..
