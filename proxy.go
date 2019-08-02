package canaryrouter

import (
	"canary-router/config"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	Main   *httputil.ReverseProxy
	Canary *httputil.ReverseProxy
}

func BuildProxies(config config.Config) (*Proxy, error) {
	urlMain, err := url.Parse(config.MainTarget)
	if err != nil {
		return nil, err
	}

	urlCanary, err := url.Parse(config.CanaryTarget)
	if err != nil {
		return nil, err
	}

	proxies := &Proxy{
		Main:   httputil.NewSingleHostReverseProxy(urlMain),
		Canary: httputil.NewSingleHostReverseProxy(urlCanary),
	}

	return proxies, nil
}
