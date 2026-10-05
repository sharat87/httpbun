package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatus(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{Path: "status/418"})
	s.Equal(http.StatusTeapot, resp.StatusCode)
	s.JSONEq(`{"code": 418, "description": "I'm a teapot"}`, body)
}

func TestStatusPlainText(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Path:    "status/418",
		Headers: map[string][]string{"Accept": {"text/plain"}},
	})
	s.Equal(http.StatusTeapot, resp.StatusCode)
	s.Equal("I'm a teapot", body)
}

func TestStatusInvalid(t *testing.T) {
	for _, path := range []string{"status/,", "status/,,", "status/100", "status/101", "status/600", "status/abc"} {
		t.Run(path, func(t *testing.T) {
			resp, _ := ExecRequest(t, R{Path: path})
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}
