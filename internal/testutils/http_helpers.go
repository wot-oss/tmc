package testutils

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
)

type Request struct {
	method  string
	route   string
	headers map[string]string
	body    []byte
}

func NewRequest(method, route string) *Request {
	return &Request{
		method:  method,
		route:   route,
		headers: make(map[string]string),
	}
}

func (r *Request) WithHeader(key, value string) *Request {
	r.headers[key] = value
	return r
}

func (r *Request) WithBody(body []byte) *Request {
	r.body = body
	return r
}

func (r *Request) RunOnHandler(h http.Handler) *httptest.ResponseRecorder {
	var reader io.Reader
	if r.body != nil {
		reader = bytes.NewReader(r.body)
	}

	req := httptest.NewRequest(r.method, r.route, reader)
	for k, v := range r.headers {
		req.Header.Add(k, v)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
