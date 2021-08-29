module github.com/darkweak/souin/plugins/caddy

go 1.15

require (
	github.com/caddyserver/caddy/v2 v2.4.4-0.20210826210025-84b906a248a7
	github.com/darkweak/souin v1.5.4
	go.uber.org/zap v1.19.0
)

replace github.com/darkweak/souin v1.5.4 => ../..
