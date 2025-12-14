package api_tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLlmProxy(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodPost,
		Path:   "llm/proxy",
		Body: `{
			"provider": "fake",
			"model": "test-model",
			"messages": [
				{"role": "user", "content": "hello"},
				{"role": "assistant", "content": "world"}
			]
		}`,
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("text/event-stream", resp.Header.Get("Content-Type"))
	s.Contains(body, `data: {"content": "HELLO"}`)
	s.Contains(body, `data: {"content": "WORLD"}`)
	s.Contains(body, "data: [DONE]")
}

func TestLlmProxyUnsupportedProvider(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodPost,
		Path:   "llm/proxy",
		Body: `{
			"provider": "unsupported",
			"model": "test-model",
			"messages": [{"role": "user", "content": "hello"}]
		}`,
	})
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.Equal("Unsupported provider: unsupported", strings.TrimSpace(body))
}

func TestLlmProxyInvalidJson(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodPost,
		Path:   "llm/proxy",
		Body:   `{invalid`,
	})
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.Contains(strings.TrimSpace(body), "Invalid JSON payload")
}

func TestLlmProxyMissingProvider(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodPost,
		Path:   "llm/proxy",
		Body: `{
			"model": "test-model",
			"messages": [{"role": "user", "content": "hello"}]
		}`,
	})
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.Equal("Missing `provider` in payload", strings.TrimSpace(body))
}

func TestLlmProxyMissingModel(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodPost,
		Path:   "llm/proxy",
		Body: `{
			"provider": "fake",
			"messages": [{"role": "user", "content": "hello"}]
		}`,
	})
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.Equal("Missing `model` in payload", strings.TrimSpace(body))
}

func TestLlmProxyMissingMessages(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodPost,
		Path:   "llm/proxy",
		Body: `{
			"provider": "fake",
			"model": "test-model"
		}`,
	})
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.Equal("Missing `messages` in payload", strings.TrimSpace(body))
}

func TestLlmProxyGetNotAllowed(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "llm/proxy",
	})
	s.Equal(http.StatusMethodNotAllowed, resp.StatusCode)
	s.Equal("Only POST is allowed", strings.TrimSpace(body))
}
