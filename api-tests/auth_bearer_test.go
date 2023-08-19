package api_tests

import (
	"github.com/sharat87/httpbun/c"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestBearerAuthWithToken(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "bearer",
		Headers: map[string][]string{
			"Authorization": {"Bearer my-auth-token"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.NotContains(resp.Header, c.WWWAuthenticate)
	s.JSONEq(`{
		"authenticated": true,
		"token": "my-auth-token"
	}`, body)
}

func TestBearerAuthWithoutToken(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "bearer",
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	s.NotContains(resp.Header, c.ContentType)
	s.Equal("Bearer", resp.Header.Get(c.WWWAuthenticate))
	s.Equal("", body)
}
