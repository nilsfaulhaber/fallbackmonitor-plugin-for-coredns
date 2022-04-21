package fallbackmonitor

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() { plugin.Register("fallbackmonitor", setupEcho) }

func setupEcho(c *caddy.Controller) error {
	handler, err := echoParse(c)
	if err != nil {
		return plugin.Error("fallbackmonitor", err)
	}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		handler.Next = next
		return handler
	})

	return nil
}

func echoParse(c *caddy.Controller) (handler Handler, err error) {
	zones := plugin.OriginsFromArgsOrServerBlock(c.RemainingArgs(), c.ServerBlockKeys)
	handler.Zones = append(handler.Zones, zones...)
	return
}
