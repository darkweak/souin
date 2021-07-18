module github.com/darkweak/souin/plugins/tyk

go 1.15

require (
	github.com/TykTechnologies/gojsonschema v0.0.0-20170222154038-dcb3e4bb7990 // indirect
	github.com/TykTechnologies/tyk v2.9.5+incompatible
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/darkweak/souin v1.5.2
	github.com/franela/goblin v0.0.0-20210519012713-85d372ac71e2 // indirect
	github.com/franela/goreq v0.0.0-20171204163338-bcd34c9993f8 // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/hashicorp/terraform v1.0.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lonelycode/go-uuid v0.0.0-20141202165402-ed3ca8a15a93 // indirect
	github.com/lonelycode/osin v0.0.0-20160423095202-da239c9dacb6 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pmylund/go-cache v2.1.0+incompatible // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/x-cray/logrus-prefixed-formatter v0.5.2 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	go.uber.org/zap v1.16.0
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
)

replace (
	github.com/darkweak/souin v1.5.2 => ../..
	github.com/hashicorp/terraform v1.0.1 => github.com/hashicorp/terraform v0.14.11
)
