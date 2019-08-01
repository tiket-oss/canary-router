package canaryrouter

import "net/http/httputil"

type Proxy struct {
	Main   *httputil.ReverseProxy
	Canary *httputil.ReverseProxy
}
