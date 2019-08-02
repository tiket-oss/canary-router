package handler

import (
	"bytes"
	canaryrouter "canary-router"
	"canary-router/config"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

func Index(config config.Config, proxies *canaryrouter.Proxy) func(http.ResponseWriter, *http.Request) {

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	client := &http.Client{Transport: tr}

	return func(w http.ResponseWriter, req *http.Request) {

		sidecarUrl, err := url.ParseRequestURI(config.SidecarUrl)
		if err != nil {
			proxies.Main.ServeHTTP(w, req)
			return
		}

		oriUrl := req.URL
		oriBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			proxies.Main.ServeHTTP(w, req)
			return
		}

		req.URL = sidecarUrl
		req.Body = ioutil.NopCloser(bytes.NewBuffer(oriBody))

		resp, err := client.Do(req)
		if err != nil {
			proxies.Main.ServeHTTP(w, req)
			return
		}

		req.URL = oriUrl
		req.Body = ioutil.NopCloser(bytes.NewBuffer(oriBody))

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
