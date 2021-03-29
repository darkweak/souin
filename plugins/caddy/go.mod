module github.com/darkweak/souin/plugins/caddy

go 1.15

require (
	github.com/caddyserver/caddy/v2 v2.3.0
	github.com/darkweak/souin v1.4.4
	github.com/dgraph-io/ristretto v0.0.3 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20200921180117-858c6e7e6b7e // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20210309074719-68d13333faf2 // indirect
	golang.org/x/tools v0.0.0-20200608174601-1b747fd94509 // indirect
)

replace github.com/darkweak/souin latest => ../..
