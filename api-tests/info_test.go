package api_tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfoOnlyExposesPrefixedEnv(t *testing.T) {
	s := assert.New(t)
	t.Setenv("HTTPBUN_INFO_REGION", "test-region")
	t.Setenv("SOME_SECRET_TOKEN", "hunter2")

	resp, body := ExecRequest(R{Path: "info"})
	s.Equal(http.StatusOK, resp.StatusCode)

	var info struct {
		Hostname string
		Env      map[string]string
	}
	s.NoError(json.Unmarshal([]byte(body), &info))
	s.NotEmpty(info.Hostname)
	s.Equal("test-region", info.Env["HTTPBUN_INFO_REGION"])
	s.NotContains(body, "hunter2")
	for name := range info.Env {
		s.Regexp("^HTTPBUN_INFO_", name)
	}
}
