module github.com/darkweak/souin/plugins/traefik/wasm

go 1.22.4

require github.com/http-wasm/http-wasm-guest-tinygo v0.4.0

replace (
	github.com/darkweak/souin v1.6.49 => ../../..
	github.com/darkweak/storages/core => ../../../../storages/core
)
