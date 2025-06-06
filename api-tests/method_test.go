package api_tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
)

func TestAllMethods(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		for _, path := range []string{"get", "post", "put", "delete"} {
			DoMethodTest(t, method, path)
		}
	}
}

func DoMethodTest(t *testing.T, method, path string) {
	t.Helper()
	s := assert.New(t)

	resp, body := ExecRequest(R{
		Method: method,
		Path:   path,
	})

	s.Equal("httpbun/", resp.Header.Get("X-Powered-By"))

	if strings.ToUpper(path) == method {
		s.Equal(http.StatusOK, resp.StatusCode)
		s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
		extraHeaders := ""
		if method != http.MethodGet && method != http.MethodDelete {
			extraHeaders = `"Content-Length": "0",`
		}
		s.JSONEq(`{
			"args": {},
			"headers": {
				`+extraHeaders+`
				"Accept-Encoding": "gzip"
			},
			"data": "",
			"files": {},
			"form": {},
			"json": null,
			"method": "`+method+`",
			"origin": "127.0.0.1",
			"url": "http://127.0.0.1:30001/`+path+`"
		}`, body)

	} else {
		s.Equal(http.StatusMethodNotAllowed, resp.StatusCode)
		s.Equal("", body)

	}
}

func TestGetNameSherlock(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "get?name=Sherlock",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"args": {
			"name": "Sherlock"
		},
		"headers": {
			"Accept-Encoding": "gzip"
		},
		"data": "",
		"files": {},
		"form": {},
		"json": null,
		"method": "GET",
		"origin": "127.0.0.1",
		"url": "http://127.0.0.1:30001/get?name=Sherlock"
	}`, body)
}

func TestGetFirstSherlockLastHolmes(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "get?first=Sherlock&last=Holmes",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"args": {
			"first": "Sherlock",
			"last":  "Holmes"
		},
		"headers": {
			"Accept-Encoding": "gzip"
		},
		"data": "",
		"files": {},
		"form": {},
		"json": null,
		"method": "GET",
		"origin": "127.0.0.1",
		"url": "http://127.0.0.1:30001/get?first=Sherlock&last=Holmes"
	}`, body)
}

func TestGetWithCustomHeader(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "get",
		Headers: map[string][]string{
			"X-Custom": {"first-custom"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"args": {},
		"headers": {
			"Accept-Encoding": "gzip",
			"X-Custom": "first-custom"
		},
		"data": "",
		"files": {},
		"form": {},
		"json": null,
		"method": "GET",
		"origin": "127.0.0.1",
		"url": "http://127.0.0.1:30001/get"
	}`, body)
}

func TestGetWithTwoCustomHeader(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "get",
		Headers: map[string][]string{
			"X-First":  {"first-custom"},
			"X-Second": {"second-custom"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"args": {},
		"headers": {
			"Accept-Encoding": "gzip",
			"X-First": "first-custom",
			"X-Second": "second-custom"
		},
		"data": "",
		"files": {},
		"form": {},
		"json": null,
		"method": "GET",
		"origin": "127.0.0.1",
		"url": "http://127.0.0.1:30001/get"
	}`, body)
}

func TestGetWithMultipleHeaderValues(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "get",
		Headers: map[string][]string{
			"X-One": {"first one", "second one"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"args": {},
		"headers": {
			"Accept-Encoding": "gzip",
			"X-One": [
				"first one",
				"second one"
			]
		},
		"data": "",
		"files": {},
		"form": {},
		"json": null,
		"method": "GET",
		"origin": "127.0.0.1",
		"url": "http://127.0.0.1:30001/get"
	}`, body)
}

func TestMethodPostWithFormBody(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodPost,
		Path:   "post",
		Body:   "answer=42",
		Headers: map[string][]string{
			"Content-Type": {"application/x-www-form-urlencoded"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"method": "POST",
		"args": {},
		"headers": {
			"Accept-Encoding": "gzip",
			"Content-Length": "9",
			"Content-Type": "application/x-www-form-urlencoded"
		},
		"form": {
			"answer": "42"
		},
		"data": "",
		"json": null,
		"files": {},
		"origin": "127.0.0.1",
		"url": "http://127.0.0.1:30001/post"
	}`, body)
}
