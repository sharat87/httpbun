package api_tests

import (
	"github.com/sharat87/httpbun/c"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestIpInXForwardedFor(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "ip",
		Headers: map[string][]string{
			"X-Httpbun-Forwarded-For": {"12.34.56.78"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"origin": "12.34.56.78"
	}`, body)
}
