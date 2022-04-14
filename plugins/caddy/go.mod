module github.com/darkweak/souin/plugins/caddy

go 1.16

require (
	github.com/caddyserver/caddy/v2 v2.4.6
	github.com/darkweak/souin v1.6.6
	go.uber.org/zap v1.21.0
)

replace github.com/darkweak/souin v1.6.6 => ../..
