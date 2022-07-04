module github.com/darkweak/souin/plugins/tyk

go 1.16

require (
	github.com/TykTechnologies/tyk v2.9.5+incompatible
	github.com/darkweak/souin v1.6.11
	github.com/patrickmn/go-cache v2.1.0+incompatible
	go.uber.org/zap v1.21.0
)

require (
	github.com/TykTechnologies/gojsonschema v0.0.0-20170222154038-dcb3e4bb7990 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/armon/go-metrics v0.4.0 // indirect
	github.com/buraksezer/consistent v0.9.0 // indirect
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/franela/goblin v0.0.0-20211003143422-0a4f594942bf // indirect
	github.com/franela/goreq v0.0.0-20171204163338-bcd34c9993f8 // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/flatbuffers v2.0.6+incompatible // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/hcl/v2 v2.13.0 // indirect
	github.com/hashicorp/memberlist v0.3.1 // indirect
	github.com/hashicorp/terraform v1.0.1 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/klauspost/compress v1.15.7 // indirect
	github.com/lonelycode/go-uuid v0.0.0-20141202165402-ed3ca8a15a93 // indirect
	github.com/lonelycode/osin v0.0.0-20160423095202-da239c9dacb6 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/miekg/dns v1.1.50 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/pmylund/go-cache v2.1.0+incompatible // indirect
	github.com/prometheus/client_golang v1.12.2 // indirect
	github.com/prometheus/common v0.35.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/x-cray/logrus-prefixed-formatter v0.5.2 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/zclconf/go-cty v1.10.0 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d // indirect
	golang.org/x/net v0.0.0-20220630215102-69896b714898 // indirect
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f // indirect
	golang.org/x/sys v0.0.0-20220702020025-31831981b65f // indirect
	golang.org/x/term v0.0.0-20220526004731-065cf7ba2467 // indirect
	golang.org/x/tools v0.1.11 // indirect
	golang.org/x/xerrors v0.0.0-20220609144429-65e65417b02f // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220630174209-ad1d48641aa7 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/darkweak/souin v1.6.11 => ../..
	github.com/hashicorp/terraform v1.0.1 => github.com/hashicorp/terraform v0.14.11
)
