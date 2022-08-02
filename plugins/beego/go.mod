module github.com/darkweak/souin/plugins/beego

go 1.16

require (
	github.com/beego/beego/v2 v2.0.4
	github.com/darkweak/souin v1.6.18
	github.com/mitchellh/mapstructure v1.5.0 // indirect
)

replace github.com/darkweak/souin v1.6.18 => ../..
