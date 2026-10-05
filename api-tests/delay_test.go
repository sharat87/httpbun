package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDelay(t *testing.T) {
	resp, body := ExecRequest(t, R{Path: "delay/0.1"})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "OK", body)
}

func TestDelayInvalid(t *testing.T) {
	for _, delay := range []string{"NaN", "-1", "301", "Inf", "abc"} {
		t.Run(delay, func(t *testing.T) {
			resp, _ := ExecRequest(t, R{Path: "delay/" + delay})
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}
