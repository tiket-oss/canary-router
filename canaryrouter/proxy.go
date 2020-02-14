package canaryrouter

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tiket-libre/canary-router/canaryrouter/config"
)

func newTransport(clientConfig config.HTTPClientConfig) *http.Transport {
	return &http.Transport{
		ResponseHeaderTimeout: time.Duration(clientConfig.Timeout) * time.Second,
		MaxIdleConns:          clientConfig.MaxIdleConns,
		IdleConnTimeout:       time.Duration(clientConfig.IdleConnTimeout) * time.Second,
		DisableCompression:    clientConfig.DisableCompression,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: clientConfig.TLS.InsecureSkipVerify},
	}
}

func newReverseProxy(target, customHost string, dumpResponse bool) (*httputil.ReverseProxy, error) {
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
	if log.IsLevelEnabled(log.DebugLevel) {
		proxy.ModifyResponse = func(res *http.Response) error {
			dumpRes, err := httputil.DumpResponse(res, dumpResponse)
			if err != nil {
				log.WithField("from", target).Infof("Failed to dump request")
			} else {
				log.WithField("from", target).Debugf("%+v", string(dumpRes))
			}

			return nil
		}
	}

	return proxy, nil
}
