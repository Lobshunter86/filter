package filter

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"

	"github.com/caddyserver/caddy"
)

func init() { plugin.Register("filter", setup) }

func setup(c *caddy.Controller) error {
	// no args yet, so no parse, error if there are more arg
	c.Next()
	if c.NextArg() {
		return plugin.Error("filter", c.ArgErr())
	}
	// can read slowIP on run time here

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Filter{Next: next}
	})
	return nil
}
