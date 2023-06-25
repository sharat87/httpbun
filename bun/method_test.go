package bun

import (
	tu "github.com/sharat87/httpbun/test_utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMethodGet(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "get",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.JSONEq(`{
		"method": "GET",
		"args": {},
		"headers": {},
		"origin": "192.0.2.1",
		"url": "http://example.com/get"
	}`, body)
}

func TestMethodGetWithCustomHeaders(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "get",
		Headers: map[string][]string{
			"X-One": {"custom header value"},
			"X-Two": {"another custom header"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.JSONEq(`{
		"method": "GET",
		"args": {},
		"headers": {
			"X-One": "custom header value",
			"X-Two": "another custom header"
		},
		"origin": "192.0.2.1",
		"url": "http://example.com/get"
	}`, body)
}

func TestMethodGetWithMultipleHeaderValues(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "get",
		Headers: map[string][]string{
			"X-One": {"custom header value", "another custom header"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.JSONEq(`{
		"method": "GET",
		"args": {},
		"headers": {
			"X-One": "custom header value,another custom header"
		},
		"origin": "192.0.2.1",
		"url": "http://example.com/get"
	}`, body)
}

func TestMethodPost(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "POST",
		Path:   "post",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.JSONEq(`{
		"method": "POST",
		"args": {},
		"form": {},
		"data": "",
		"headers": {},
		"json": null,
		"files": {},
		"origin": "192.0.2.1",
		"url": "http://example.com/post"
	}`, body)
}

func TestMethodPostWithPlainBody(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "POST",
		Path:   "post",
		Body:   "answer=42",
		Headers: map[string][]string{
			"Content-Type": {"application/x-www-form-urlencoded"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.JSONEq(`{
		"method": "POST",
		"args": {},
		"form": {
			"answer": "42"
		},
		"data": "",
		"headers": {
			"Content-Type": "application/x-www-form-urlencoded"
		},
		"json": null,
		"files": {},
		"origin": "192.0.2.1",
		"url": "http://example.com/post"
	}`, body)
}
