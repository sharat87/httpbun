package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
)

func TestHeaders(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "headers",
		Headers: map[string][]string{
			"X-One": {"custom header value"},
			"X-Two": {"another custom header"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"headers": {
			"Accept-Encoding": "gzip",
			"X-One": "custom header value",
			"X-Two": "another custom header"
		}
	}`, body)
}

func TestHeadersRepeat(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "headers",
		Headers: map[string][]string{
			"X-One": {"custom header value", "another custom header"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"headers": {
			"Accept-Encoding": "gzip",
			"X-One": [
				"custom header value",
				"another custom header"
			]
		}
	}`, body)
}
