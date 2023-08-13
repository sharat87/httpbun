package bun

import (
	tu "github.com/sharat87/httpbun/test_utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRedirectTo(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "redirect?url=http://target-url",
	})
	s.Equal(302, resp.StatusCode)
	s.Equal("http://target-url", resp.Header.Get("Location"))
}

func TestRedirectToWithEncodedURL(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "redirect?url=http%3A%2F%2F" + "target-url",
	})
	s.Equal(302, resp.StatusCode)
	s.Equal("http://target-url", resp.Header.Get("Location"))
}

func TestRedirectToWithURLAndStatus(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "redirect?url=http://target-url&status=301",
	})
	s.Equal(301, resp.StatusCode)
	s.Equal("http://target-url", resp.Header.Get("Location"))
}

func TestRedirectToWithoutURL(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "redirect",
	})
	s.Equal(400, resp.StatusCode)
	s.Equal("Need url parameter\n", body)
}

func TestRedirectRelative(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "redirect/4",
	})
	s.Equal(302, resp.StatusCode)
	s.Equal("3", resp.Header.Get("Location"))
}

func TestRedirectTooHigh(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "redirect/40",
	})
	s.Equal(400, resp.StatusCode)
	s.Equal("", resp.Header.Get("Location"))
}

func TestRedirectRelative2(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "relative-redirect/4",
	})
	s.Equal(302, resp.StatusCode)
	s.Equal("3", resp.Header.Get("Location"))
}

func TestRedirectAbsolute(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "absolute-redirect/4",
	})
	s.Equal(302, resp.StatusCode)
	s.Equal("/absolute-redirect/3", resp.Header.Get("Location"))
}
