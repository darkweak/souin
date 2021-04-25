module github.com/darkweak/souin/plugins/caddy

go 1.15

require (
	github.com/caddyserver/caddy/v2 v2.3.0
	github.com/darkweak/souin v1.5.0
	github.com/dgraph-io/ristretto v0.0.3 // indirect
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	go.uber.org/zap v1.16.0
)

replace github.com/darkweak/souin v1.5.0 => ../..
