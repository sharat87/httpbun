package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
)

func TestResponseHeaders(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "response-headers?one=two&three=four",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.Equal([]string{"two"}, resp.Header.Values("One"))
	s.Equal([]string{"four"}, resp.Header.Values("Three"))
	s.JSONEq(`{
		"responseHeaders": {
			"Content-Length": "138",
			"Content-Type": "application/json",
			"One": "two",
			"Three": "four"
		}
	}`, body)
}

func TestResponseHeadersRepeated(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "response-headers?one=two&one=four",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.Equal([]string{"two", "four"}, resp.Header.Values("One"))
	s.JSONEq(`{
		"responseHeaders": {
			"Content-Length": "145",
			"Content-Type": "application/json",
			"One": ["two", "four"]
		}
	}`, body)
}

func TestResponseHeadersRejectsExternalLocation(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "response-headers?Location=https://target-url",
	})
	s.Equal(http.StatusForbidden, resp.StatusCode)
	s.Empty(resp.Header.Get(c.Location))
	s.Equal("Forbidden redirect URL. Please be careful with this link.", body)
}
