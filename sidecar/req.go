package sidecar

import "net/http"

type OriginRequest struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Header http.Header `json:"header"`
	Body   string      `json:"body"`
}
