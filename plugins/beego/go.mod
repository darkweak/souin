module github.com/darkweak/souin/plugins/beego

go 1.16

require (
	github.com/beego/bee/v2 v2.0.2 // indirect
	github.com/beego/beego v1.12.8
	github.com/beego/beego/v2 v2.0.3
	github.com/bnkamalesh/webgo/v6 v6.6.1
	github.com/darkweak/souin v1.6.7
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	go.uber.org/zap v1.21.0
)

replace github.com/darkweak/souin v1.6.7 => ../..
