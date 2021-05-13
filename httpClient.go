package main

import (
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

func (b *BiliroamingGo) initProxy(c *Config) (*fasthttp.Client, *fasthttp.Client, *fasthttp.Client, *fasthttp.Client, *fasthttp.Client) {
	cnClient := b.newClient(c.ProxyCN)
	hkClient := b.newClient(c.ProxyHK)
	twClient := b.newClient(c.ProxyTW)
	thClient := b.newClient(c.ProxyTH)
	defaultClient := &fasthttp.Client{}
	return cnClient, hkClient, twClient, thClient, defaultClient
}

func (b *BiliroamingGo) newClient(proxy string) *fasthttp.Client {
	if proxy != "" {
		b.sugar.Debug("New socks proxy client: ", proxy)
		return &fasthttp.Client{
			Dial: fasthttpproxy.FasthttpSocksDialer(proxy),
		}
	}
	b.sugar.Debug("New normal client")
	return &fasthttp.Client{}
}

func (b *BiliroamingGo) getClientByArea(area string) *fasthttp.Client {
	switch strings.ToLower(area) {
	case "cn":
		return b.cnClient
	case "hk":
		return b.hkClient
	case "tw":
		return b.twClient
	case "th":
		return b.thClient
	default:
		return b.defaultClient
	}
}

func (b *BiliroamingGo) getReverseProxyByArea(area string) string {
	switch strings.ToLower(area) {
	case "cn":
		return b.config.ReverseCN
	case "hk":
		return b.config.ReverseHK
	case "tw":
		return b.config.ReverseTW
	case "th":
		return b.config.ReverseTH
	default:
		return ""
	}
}
