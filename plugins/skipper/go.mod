module github.com/darkweak/souin/plugins/skipper

go 1.16

require (
	github.com/darkweak/souin v1.6.3
	github.com/zalando/skipper v0.13.174
	go.uber.org/zap v1.19.1
)

replace github.com/darkweak/souin v1.6.3 => ../..
