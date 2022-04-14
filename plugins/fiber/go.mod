module github.com/darkweak/souin/plugins/fiber

go 1.16

require (
	github.com/darkweak/souin v1.6.6
	github.com/gofiber/fiber/v2 v2.31.0
	github.com/valyala/fasthttp v1.35.0
	go.uber.org/zap v1.21.0
)

replace github.com/darkweak/souin v1.6.6 => ../..
