module github.com/darkweak/souin/plugins/traefik

go 1.16

replace github.com/darkweak/souin v1.5.4-beta1 => ../..

require (
	github.com/darkweak/souin v1.5.4-beta1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	go.uber.org/zap v1.19.0
	golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e // indirect
)
