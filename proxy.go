package canaryrouter

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/tiket-libre/canary-router/config"

	"github.com/juju/errors"
)

// Proxy holds the reference to instance of Main and Canary httputil.ReverseProxy
// that is going to be used to route traffic
type Proxy struct {
	Main   *httputil.ReverseProxy
	Canary *httputil.ReverseProxy
}

// BuildProxies constructs a Proxy object with mainTargetURL as the URL for Main proxy
// and canaryTargetURL as the URL for Canary proxy
func BuildProxies(configClient config.HTTPClientConfig, mainTargetURL, canaryTargetURL string) (*Proxy, error) {

	proxyMain, err := newReverseProxy(mainTargetURL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	proxyMain.Transport = newTransport(configClient.MaxIdleConns, configClient.IdleConnTimeout, configClient.DisableCompression)

	proxyCanary, err := newReverseProxy(canaryTargetURL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	proxyCanary.Transport = newTransport(configClient.MaxIdleConns, configClient.IdleConnTimeout, configClient.DisableCompression)

	proxies := &Proxy{
		Main:   proxyMain,
		Canary: proxyCanary,
	}

	return proxies, nil
}

func newTransport(maxIdleConns, idleConnTimeout int, disableCompression bool) *http.Transport {
	return &http.Transport{
		MaxIdleConns:       maxIdleConns,
		IdleConnTimeout:    time.Duration(idleConnTimeout) * time.Second,
		DisableCompression: disableCompression,
	}
}

func newReverseProxy(target string) (*httputil.ReverseProxy, error) {
	url, err := url.ParseRequestURI(target)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return httputil.NewSingleHostReverseProxy(url), nil
}
