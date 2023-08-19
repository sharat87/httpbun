package api_tests

import (
	"github.com/sharat87/httpbun/c"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestRedirectTo(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect?url=http://target-url",
	})
	s.Equal(http.StatusFound, resp.StatusCode)
	s.Equal("http://target-url", resp.Header.Get(c.Location))
}

func TestRedirectToWithEncodedURL(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect?url=http%3A%2F%2F" + "target-url",
	})
	s.Equal(http.StatusFound, resp.StatusCode)
	s.Equal("http://target-url", resp.Header.Get(c.Location))
}

func TestRedirectToWithStatus(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(R{
		Path: "redirect?url=http://target-url&status=301",
	})
	s.Equal(http.StatusMovedPermanently, resp.StatusCode)
	s.Equal("http://target-url", resp.Header.Get(c.Location))
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
