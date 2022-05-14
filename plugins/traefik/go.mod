module github.com/darkweak/souin/plugins/traefik

go 1.16

require (
	github.com/darkweak/souin v1.6.8
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pquerna/cachecontrol v0.1.0
	go.uber.org/zap v1.21.0
)

replace github.com/darkweak/souin v1.6.8 => ../..
