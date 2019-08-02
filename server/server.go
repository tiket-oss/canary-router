package server

import (
	canaryrouter "canary-router"
	"canary-router/config"
	"canary-router/handler"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func Run(config config.Config) error {

	urlMain, err := url.Parse(config.MainTarget)
	if err != nil {
		return err
	}

	urlCanary, err := url.Parse(config.CanaryTarget)
	if err != nil {
		return err
	}

	proxies := canaryrouter.Proxy{
		Main:   httputil.NewSingleHostReverseProxy(urlMain),
		Canary: httputil.NewSingleHostReverseProxy(urlCanary),
	}

	http.HandleFunc("/", handler.Index(config, proxies))

	return http.ListenAndServe(fmt.Sprintf(":%d", config.ListenPort), nil)
}
