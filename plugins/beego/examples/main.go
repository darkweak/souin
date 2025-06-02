package main

import (
	_ "github.com/beego/beego/v2/core/config/json"
	"github.com/beego/beego/v2/server/web"
	httpcache "github.com/darkweak/souin/plugins/beego"
)

type mainController struct {
	web.Controller
}

func (c *mainController) Get() {
	c.Ctx.WriteString("hello world" + c.Ctx.Request.URL.Path)
}

func init() {
}

func main() {
	_ = web.LoadAppConfig("json", "beego.json")
	web.InsertFilterChain("/*", httpcache.NewHTTPCacheFilter())
	web.Router("/*", &mainController{})
	web.Run()
}
