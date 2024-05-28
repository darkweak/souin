# tscert

This is a stripped down version of the
`tailscale.com/client/tailscale` Go package but with minimal
dependencies and supporting older versions of Go.

It's meant for use by Caddy, so they don't need to depend on Go 1.17 yet.
Also, it has the nice side effect of not polluting their `go.sum` file
because `tailscale.com` is a somewhat large module.

## Docs

See https://pkg.go.dev/github.com/tailscale/tscert
