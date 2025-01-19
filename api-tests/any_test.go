package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
)

func TestAnythingMethods(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		DoAnythingTest(t, method)
	}
}

func DoAnythingTest(t *testing.T, method string) {
	t.Helper()
	s := assert.New(t)

	resp, body := ExecRequest(R{
		Method: method,
		Path:   "any",
	})

	s.Equal("httpbun/", resp.Header.Get("X-Powered-By"))

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
		"url": "http://127.0.0.1:30001/any"
	}`, body)

}

func TestAnythingWithExtraPath(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "any/some-random-path-stuff-here",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"args": {},
		"headers": {
			"Accept-Encoding": "gzip"
		},
		"data": "",
		"files": {},
		"form": {},
		"json": null,
		"method": "GET",
		"origin": "127.0.0.1",
		"url": "http://127.0.0.1:30001/any/some-random-path-stuff-here"
	}`, body)
}

func TestAnythingWithExtraPathInvalid(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "anymore",
	})
	s.Equal(http.StatusNotFound, resp.StatusCode)
	s.Equal(c.TextPlain, resp.Header.Get(c.ContentType))
	s.Equal("404 page not found\n", body)
}

func TestAnythingWithQueryParams(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "any?name=Sherlock",
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
		"url": "http://127.0.0.1:30001/any?name=Sherlock"
	}`, body)
}

func TestAnythingFirstSherlockLastHolmes(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "any?first=Sherlock&last=Holmes",
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
		"url": "http://127.0.0.1:30001/any?first=Sherlock&last=Holmes"
	}`, body)
}

func TestAnyWithCustomHeader(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "any",
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
		"url": "http://127.0.0.1:30001/any"
	}`, body)
}

func TestAnythingWithTwoCustomHeader(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "any",
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
		"url": "http://127.0.0.1:30001/any"
	}`, body)
}

func TestAnyWithMultipleHeaderValues(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "any",
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
		"url": "http://127.0.0.1:30001/any"
	}`, body)
}

func TestAnythingWithFormBody(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodPost,
		Path:   "any",
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
		"url": "http://127.0.0.1:30001/any"
	}`, body)
}
