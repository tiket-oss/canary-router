package canaryrouter

const (
	// StatusCodeMain is the expected HTTP status code from sidecar service which will
	// route traffic to Main proxy
	StatusCodeMain = 204

	// StatusCodeCanary is the expected HTTP status code from sidecar service which will
	// route traffic to Canary proxy
	StatusCodeCanary = 200
)
