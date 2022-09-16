module github.com/darkweak/souin/plugins/tyk

go 1.16

require (
	github.com/TykTechnologies/tyk v1.9.2-0.20220711082452-7b07f4c2fd27
	github.com/darkweak/souin v1.6.18
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pquerna/cachecontrol v0.1.0
	go.uber.org/zap v1.21.0
)

replace github.com/darkweak/souin v1.6.18 => ../..
