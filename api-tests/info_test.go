package api_tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/server/spec"
)

func TestInfo(t *testing.T) {
	s := assert.New(t)
	srv := NewServer(t, spec.Spec{InfoEnv: map[string]string{"HTTPBUN_INFO_REGION": "test-region"}})

	resp, body := srv.Exec(t, R{Path: "info"})
	s.Equal(http.StatusOK, resp.StatusCode)

	var info struct {
		Hostname string
		Env      map[string]string
	}
	s.NoError(json.Unmarshal([]byte(body), &info))
	s.NotEmpty(info.Hostname)
	s.Equal(map[string]string{"HTTPBUN_INFO_REGION": "test-region"}, info.Env)
}
