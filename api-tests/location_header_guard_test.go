package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
)

func TestMixRedirectAllowsApprovedDomain(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "mix/r=https%3A%2F%2Fexample.com",
	})
	s.Equal(http.StatusTemporaryRedirect, resp.StatusCode)
	s.Equal("https://example.com", resp.Header.Get(c.Location))
}

func TestMixRedirectRejectsUnknownDomain(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/r=https%3A%2F%2Ftarget-url",
	})
	s.Equal(http.StatusForbidden, resp.StatusCode)
	s.Empty(resp.Header.Get(c.Location))
	s.Equal("Forbidden redirect URL. Please be careful with this link.", body)
}

func TestMixHeaderRedirectRejectsUnknownDomain(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "mix/s=301/h=location:https%3A%2F%2Ftarget-url",
	})
	s.Equal(http.StatusForbidden, resp.StatusCode)
	s.Empty(resp.Header.Get(c.Location))
	s.Equal("Forbidden redirect URL. Please be careful with this link.", body)
}
