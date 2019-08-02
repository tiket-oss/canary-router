package handler

import (
	canaryrouter "canary-router"
	"canary-router/config"
	"net/http"
	"net/url"
	"time"
)

//func Index(w http.ResponseWriter, r *http.Request) {
func Index(config config.Config, proxies canaryrouter.Proxy) func(http.ResponseWriter, *http.Request) {

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	client := &http.Client{Transport: tr}

	return func(w http.ResponseWriter, req *http.Request) {

		sidecarUrl, err := url.Parse(config.SidecarUrl)
		if err != nil {
			proxies.Main.ServeHTTP(w, req)
			return
		}

		req.URL = sidecarUrl

		resp, err := client.Do(req)
		if err != nil {
			proxies.Main.ServeHTTP(w, req)
			return
		}

		switch resp.StatusCode {
		case canaryrouter.StatusCodeMain:
			proxies.Main.ServeHTTP(w, req)
		case canaryrouter.StatusCodeCanary:
			proxies.Canary.ServeHTTP(w, req)
		default:
			proxies.Main.ServeHTTP(w, req)
		}

		return
	}
}
