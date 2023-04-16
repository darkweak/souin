module github.com/darkweak/souin/plugins/tyk

go 1.16

require (
	github.com/Shopify/sarama v1.38.1 // indirect
	github.com/TykTechnologies/gojsonschema v0.0.0-20221026223418-8ec6134c8a60 // indirect
	github.com/TykTechnologies/tyk v1.9.2-0.20230330071232-370295d796b5
	github.com/darkweak/souin v1.6.36
	github.com/eclipse/paho.mqtt.golang v1.4.2 // indirect
	github.com/evanphx/json-patch/v5 v5.5.0 // indirect
	github.com/go-test/deep v1.0.8 // indirect
	github.com/gobwas/ws v1.2.0 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/klauspost/compress v1.16.4 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/miekg/dns v1.1.53 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/nats-io/nats.go v1.25.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pquerna/cachecontrol v0.1.1-0.20230415224848-baaf0ee61529
	github.com/r3labs/sse/v2 v2.10.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/ugorji/go v1.2.7 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/goleak v1.1.12 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.24.0
	golang.org/x/crypto v0.8.0 // indirect
	golang.org/x/tools v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
)

replace (
	github.com/darkweak/souin v1.6.36 => ../..
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 => github.com/alecthomas/kingpin/v2 v2.3.2
)
