package api_tests

import (
	"github.com/sharat87/httpbun/c"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
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
		"Content-Length": "103",
		"Content-Type": "application/json",
		"One": "two",
		"Three": "four"
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
		"Content-Length": "106",
		"Content-Type": "application/json",
		"One": ["two", "four"]
	}`, body)
}
