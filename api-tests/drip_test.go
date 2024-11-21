package api_tests

import (
	"github.com/sharat87/httpbun/c"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"
)

// todo: test for drip timing as well

func TestDrip(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "drip?duration=1&delay=0",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("text/octet-stream", resp.Header.Get(c.ContentType))
	s.Equal(strings.Repeat("*", 10), body)
}
