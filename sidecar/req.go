package sidecar

import "net/http"

// OriginRequest is a wrapper for received request to be passed on to Sidecar service
type OriginRequest struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Header http.Header `json:"header"`
	Body   string      `json:"body"`
}
