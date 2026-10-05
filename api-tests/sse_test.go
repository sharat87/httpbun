package api_tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
)

func TestSSE(t *testing.T) {
	s := assert.New(t)
	start := time.Now()
	resp, body := ExecRequest(R{Path: "sse?count=2&delay=1"})
	elapsed := time.Since(start)
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("text/event-stream", resp.Header.Get(c.ContentType))
	s.Equal(2, strings.Count(body, "event: ping"))
	s.Contains(body, "id: 2\n")
	// One delay between the two events, and none after the last one.
	s.Less(elapsed, 1900*time.Millisecond)
}

func TestSSELimits(t *testing.T) {
	for _, query := range []string{"count=0", "count=101", "delay=0", "delay=11"} {
		t.Run(query, func(t *testing.T) {
			resp, _ := ExecRequest(R{Path: "sse?" + query})
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}
