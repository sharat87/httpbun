package api_tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
)

// todo: test for drip timing as well

func TestDrip(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "drip?duration=1&delay=0",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("application/octet-stream", resp.Header.Get(c.ContentType))
	s.Equal(strings.Repeat("*", 10), body)
}

func TestDripWithCode(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "drip-lines?duration=0&delay=0&numbytes=2&code=503",
	})
	s.Equal(http.StatusServiceUnavailable, resp.StatusCode)
	s.Equal("*\n*\n", body)
}

func TestDripWithInvalidCode(t *testing.T) {
	for _, code := range []string{"100", "999"} {
		t.Run(code, func(t *testing.T) {
			resp, _ := ExecRequest(R{Path: "drip?delay=0&code=" + code})
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestDripWithDecimalSeconds(t *testing.T) {
	s := assert.New(t)
	start := time.Now()
	resp, body := ExecRequest(R{Path: "drip?delay=0.25&duration=.5&numbytes=5"})
	elapsed := time.Since(start)
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("*****", body)
	s.GreaterOrEqual(elapsed, 700*time.Millisecond)
	s.Less(elapsed, 1500*time.Millisecond)
}

func TestDripWithInvalidSeconds(t *testing.T) {
	for _, query := range []string{"delay=0.125", "duration=-1", "delay=abc", "duration=1e3", "delay=1."} {
		t.Run(query, func(t *testing.T) {
			resp, _ := ExecRequest(R{Path: "drip?" + query})
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}
