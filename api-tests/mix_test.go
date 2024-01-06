package api_tests

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"
)

func TestMixStatus(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "mix/s=200",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("", body)
	resp, body = ExecRequest(R{
		Method: http.MethodPost,
		Path:   "mix/s=200",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("", body)
}

func TestMixHeaders(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/h=x-key:val%2fmore",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("val/more", resp.Header.Get("X-Key"))
	s.Equal("", body)
}

func TestMixHeaders2(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/h=x-key:val/h=x-key2:val2",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("val", resp.Header.Get("X-Key"))
	s.Equal("val2", resp.Header.Get("X-Key2"))
	s.Equal("", body)
}

func TestMixCookies(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/c=name:content",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("name=content; Path=/", resp.Header.Get("Set-Cookie"))
	s.Equal("", body)
}

func TestMixCookies2(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/c=name:content/c=another:more",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.ElementsMatch([]string{"name=content; Path=/", "another=more; Path=/"}, resp.Header.Values("Set-Cookie"))
	s.Equal("", body)
}

func TestMixDeleteCookie(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/cd=name",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("name=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0", resp.Header.Get("Set-Cookie"))
	s.Equal("", body)
}

func TestMixDeleteCookie2(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/cd=name/cd=another",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.ElementsMatch([]string{
		"name=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0",
		"another=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0",
	}, resp.Header.Values("Set-Cookie"))
	s.Equal("", body)
}

func TestMixRedirect(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/r=http%3A%2F%2Fexample.com",
	})
	s.Equal(http.StatusTemporaryRedirect, resp.StatusCode)
	s.Equal("http://example.com", resp.Header.Get("Location"))
	s.Equal("", body)
}

func TestMixRedirect2(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/r=http%3A%2F%2Fexample.com/r=another",
	})
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.Equal("", resp.Header.Get("Location"))
	s.Equal("multiple redirects not allowed", strings.TrimSpace(body))
}

func TestMixRedirectWithStatus(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/s=301/r=http%3A%2F%2Fexample.com",
	})
	s.Equal(301, resp.StatusCode)
	s.Equal("http://example.com", resp.Header.Get("Location"))
	s.Equal("", strings.TrimSpace(body))
}

func TestMixBody(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/b64=c2FtcGxl",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("sample", body)
}
