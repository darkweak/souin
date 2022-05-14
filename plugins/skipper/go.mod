module github.com/darkweak/souin/plugins/skipper

go 1.16

require (
	github.com/darkweak/souin v1.6.8
	github.com/zalando/skipper v0.13.174
	go.uber.org/zap v1.21.0
)

replace github.com/darkweak/souin v1.6.8 => ../..
