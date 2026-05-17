package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
)

func TestRedirectTo(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect?url=https://example.com",
	})
	s.Equal(http.StatusFound, resp.StatusCode)
	s.Equal("https://example.com", resp.Header.Get(c.Location))
}

func TestRedirectToWithEncodedURL(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect?url=https%3A%2F%2Fexample.com",
	})
	s.Equal(http.StatusFound, resp.StatusCode)
	s.Equal("https://example.com", resp.Header.Get(c.Location))
}

func TestRedirectToWithStatus(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect?url=https://example.com&status=301",
	})
	s.Equal(http.StatusMovedPermanently, resp.StatusCode)
	s.Equal("https://example.com", resp.Header.Get(c.Location))
}

func TestRedirectToRejectsUnknownDomain(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "redirect?url=https://target-url",
	})
	s.Equal(http.StatusForbidden, resp.StatusCode)
	s.Empty(resp.Header.Get(c.Location))
	s.Equal("Forbidden redirect URL. Please be careful with this link.", body)
}

func TestRedirectToRejectsSchemeRelativeURL(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "redirect?url=%2F%2Fevil.example",
	})
	s.Equal(http.StatusForbidden, resp.StatusCode)
	s.Empty(resp.Header.Get(c.Location))
	s.Equal("Forbidden redirect URL. Please be careful with this link.", body)
}

func TestRedirectToWithoutURL(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect",
	})
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.NotContains(resp.Header, c.Location)
}

func TestRedirectRelative4(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect/4",
	})
	s.Equal(http.StatusFound, resp.StatusCode)
	s.Equal("3", resp.Header.Get(c.Location))
}

func TestRedirectRelative1(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect/1",
	})
	s.Equal(http.StatusFound, resp.StatusCode)
	s.Equal("../anything", resp.Header.Get(c.Location))
}

func TestRedirectCountTooHigh(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect/40",
	})
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.NotContains(resp.Header, c.Location)
}

func TestRedirectNegativeCount(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect/-5",
	})
	s.NotEqual(http.StatusFound, resp.StatusCode)
	s.NotContains(resp.Header, c.Location)
}

func TestRedirectExplicitRelative4(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "relative-redirect/4",
	})
	s.Equal(http.StatusFound, resp.StatusCode)
	s.Equal("3", resp.Header.Get(c.Location))
}

func TestRedirectAbsolute4(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "absolute-redirect/4",
	})
	s.Equal(http.StatusFound, resp.StatusCode)
	s.Equal("/absolute-redirect/3", resp.Header.Get(c.Location))
}

func TestRedirectAbsolute1(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "absolute-redirect/1",
	})
	s.Equal(http.StatusFound, resp.StatusCode)
	s.Equal("/anything", resp.Header.Get(c.Location))
}
