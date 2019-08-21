package canaryrouter

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/tiket-libre/canary-router/config"

	"github.com/juju/errors"
)

// Proxy holds the reference to instance of Main and Canary httputil.ReverseProxy
// that is going to be used to route traffi	c
type Proxy struct { // TODO: get a better name or combine with server
	Main   *httputil.ReverseProxy
	Canary *httputil.ReverseProxy
}

// BuildProxies constructs a Proxy object with mainTargetURL as the URL for Main proxy
// and canaryTargetURL as the URL for Canary proxy
func BuildProxies(configClient config.HTTPClientConfig, mainTargetURL, mainHeaderHost, canaryTargetURL, canaryHeaderHost string) (*Proxy, error) {

	proxyMain, err := newReverseProxy(mainTargetURL, mainHeaderHost)
	if err != nil {
		return nil, errors.Trace(err)
	}
	proxyMain.Transport = newTransport(configClient.MaxIdleConns, configClient.IdleConnTimeout, configClient.DisableCompression, configClient.TLS)
	proxyMain.ErrorLog = log.New(os.Stderr, "[proxy-main] ", log.LstdFlags|log.Llongfile)

	proxyCanary, err := newReverseProxy(canaryTargetURL, canaryHeaderHost)
	if err != nil {
		return nil, errors.Trace(err)
	}
	proxyCanary.Transport = newTransport(configClient.MaxIdleConns, configClient.IdleConnTimeout, configClient.DisableCompression, configClient.TLS)
	proxyCanary.ErrorLog = log.New(os.Stderr, "[proxy-canary] ", log.LstdFlags|log.Llongfile)

	proxies := &Proxy{
		Main:   proxyMain,
		Canary: proxyCanary,
	}

	return proxies, nil
}

func newTransport(maxIdleConns, idleConnTimeout int, disableCompression bool, tlsConfig config.TLS) *http.Transport {
	return &http.Transport{
		MaxIdleConns:       maxIdleConns,
		IdleConnTimeout:    time.Duration(idleConnTimeout) * time.Second,
		DisableCompression: disableCompression,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: tlsConfig.InsecureSkipVerify},
	}
}

func newReverseProxy(target, customHost string) (*httputil.ReverseProxy, error) {
	url, err := url.ParseRequestURI(target)
	if err != nil {
		return nil, errors.Trace(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		if customHost != "" {
			req.Host = customHost
		} else {
			req.Host = req.URL.Host
		}
	}

	return proxy, nil
}
