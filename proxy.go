package canaryrouter

import (
	"net/http/httputil"
	"net/url"

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
func BuildProxies(mainTargetURL, canaryTargetURL string) (*Proxy, error) {
	proxyMain, err := newReverseProxy(mainTargetURL)
	if err != nil {
		return nil, errors.Trace(err)
	}

	proxyCanary, err := newReverseProxy(canaryTargetURL)
	if err != nil {
		return nil, errors.Trace(err)
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
		return nil, errors.Trace(err)
	}

	return httputil.NewSingleHostReverseProxy(url), nil
}
