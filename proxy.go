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
	proxyMain, err := newReverseProxy(config.MainTarget)
	if err != nil {
		return nil, err
	}

	proxyCanary, err := newReverseProxy(config.CanaryTarget)
	if err != nil {
		return nil, err
	}

	proxies := &Proxy{
		Main:   proxyMain,
		Canary: proxyCanary,
	}

	return proxies, nil
}

func newReverseProxy(target string) (*httputil.ReverseProxy, error) {
	url, err := url.ParseRequestURI(target)
	if err != nil {
		return nil, err
	}

	return httputil.NewSingleHostReverseProxy(url), nil
}
