package server

import (
	"canary-router/config"
	"canary-router/handler"
	"canary-router"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func Run() error  {

	urlMain, err := url.Parse(config.GlobalConfig.MainTarget)
	if err != nil {
		return err
	}

	urlCanary, err := url.Parse(config.GlobalConfig.CanaryTarget)
	if err != nil {
		return err
	}

	proxies := canaryrouter.Proxy{
		Main: httputil.NewSingleHostReverseProxy(urlMain),
		Canary: httputil.NewSingleHostReverseProxy(urlCanary),
	}

	http.HandleFunc("/", handler.Index(proxies))

	return http.ListenAndServe(fmt.Sprintf(":%d", config.GlobalConfig.ListenPort), nil)
}

