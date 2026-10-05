package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytes(t *testing.T) {
	s := assert.New(t)
	// The test server has no size limit configured, so only zero bytes is allowed.
	resp, body := ExecRequest(R{Path: "bytes/0"})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Empty(body)

	resp, _ = ExecRequest(R{Path: "bytes/-1"})
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}
