module github.com/darkweak/souin/plugins/webgo

go 1.16

require (
	github.com/bnkamalesh/webgo/v6 v6.6.1
	github.com/darkweak/souin v1.6.6
	go.uber.org/zap v1.21.0
)

replace github.com/darkweak/souin v1.6.6 => ../..
