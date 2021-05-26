module github.com/caddyserver/cache-handler

go 1.15

require (
	github.com/buraksezer/olric v0.3.7
	github.com/caddyserver/caddy/v2 v2.4.0-beta.2
	github.com/darkweak/souin v1.5.1
	github.com/dgraph-io/ristretto v0.0.3 // indirect
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	go.uber.org/zap v1.16.0
)

replace github.com/darkweak/souin v1.5.1 => ../..
