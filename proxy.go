package canaryrouter

import (
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	Main   *httputil.ReverseProxy
	Canary *httputil.ReverseProxy
}

func BuildProxies(mainTargetUrl, canaryTargetUrl string) (*Proxy, error) {
	proxyMain, err := newReverseProxy(mainTargetUrl)
	if err != nil {
		return nil, err
	}

	proxyCanary, err := newReverseProxy(canaryTargetUrl)
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
