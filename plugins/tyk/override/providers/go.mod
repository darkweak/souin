module github.com/darkweak/souin/plugins/tyk/override/providers

go 1.15

replace github.com/darkweak/souin v1.5.2 => ../../../..

require (
	github.com/buraksezer/olric v0.3.11
	github.com/darkweak/souin v1.5.2
	github.com/google/uuid v1.3.0
	go.uber.org/zap v1.19.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)
