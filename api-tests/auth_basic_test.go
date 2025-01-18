package api_tests

import (
	"encoding/base64"
	"github.com/sharat87/httpbun/c"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestBasicAuthSuccess(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "basic-auth/scott/tiger",
		Headers: map[string][]string{
			"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("scott:tiger"))},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.NotContains(resp.Header, c.WWWAuthenticate)
	s.JSONEq(`{
		"authenticated": true,
		"user": "scott"
	}`, body)
}

func TestBasicAuthIncorrectPassword(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "basic-auth/scott/tiger",
		Headers: map[string][]string{
			"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("scott:incorrect"))},
		},
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	s.NotContains(resp.Header, c.ContentType)
	s.Equal("Basic realm=\"httpbun realm\"", resp.Header.Get(c.WWWAuthenticate))
	s.Equal("", body)
}

func TestBasicAuthIncorrectUsername(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "basic-auth/scott/tiger",
		Headers: map[string][]string{
			"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("tom:tiger"))},
		},
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	s.NotContains(resp.Header, c.ContentType)
	s.Equal("Basic realm=\"httpbun realm\"", resp.Header.Get(c.WWWAuthenticate))
	s.Equal("", body)
}

func TestBasicAuthIncorrectCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "basic-auth/scott/tiger",
		Headers: map[string][]string{
			"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("tom:lion"))},
		},
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	s.NotContains(resp.Header, c.ContentType)
	s.Equal("Basic realm=\"httpbun realm\"", resp.Header.Get(c.WWWAuthenticate))
	s.Equal("", body)
}

func TestBasicAuthMissingCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "basic-auth/scott/tiger",
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	s.NotContains(resp.Header, c.ContentType)
	s.Equal("Basic realm=\"httpbun realm\"", resp.Header.Get(c.WWWAuthenticate))
	s.Equal("", body)
}
