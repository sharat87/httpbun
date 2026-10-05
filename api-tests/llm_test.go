package api_tests

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLLMAnthropicMessagesWithoutV1(t *testing.T) {
	for _, path := range []string{"llm/v1/messages", "llm/messages"} {
		t.Run(path, func(t *testing.T) {
			resp, body := ExecRequest(t, R{
				Method: http.MethodPost,
				Path:   path,
				Body:   `{"max_tokens": 10, "messages": [{"role": "user", "content": "Hi"}]}`,
			})
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Contains(t, body, `"type": "message"`)
		})
	}
}

func TestLLMChatCompletionsWithContentParts(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Method: http.MethodPost,
		Path:   "llm/v1/chat/completions",
		Body:   `{"messages": [{"role": "user", "content": [{"type": "text", "text": "Hi there"}]}]}`,
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))

	var result struct {
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
		} `json:"usage"`
	}
	if s.NoError(json.Unmarshal([]byte(body), &result), "body: %s", body) {
		s.Equal(4, result.Usage.PromptTokens) // "user: Hi there\n"
	}
}

func TestLLMCompletionsWithCustomContent(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Method: http.MethodPost,
		Path:   "llm/v1/completions",
		Body:   `{"prompt": "hi", "httpbun": {"content": "custom"}}`,
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))

	var result struct {
		Choices []struct {
			Text string `json:"text"`
		} `json:"choices"`
	}
	if s.NoError(json.Unmarshal([]byte(body), &result), "body: %s", body) && s.Len(result.Choices, 1) {
		s.Equal("custom", result.Choices[0].Text)
	}
}

func TestLLMLargeRequestBody(t *testing.T) {
	// The body size limit for LLM endpoints, `maxBodySize` in the llm package.
	const maxBodySize = 1 << 20

	message := func(size int) string {
		return `{"messages": [{"role": "user", "content": "` + strings.Repeat("a", size) + `"}]}`
	}

	// Larger than the 10KB limit for other endpoints.
	for _, path := range []string{"llm/v1/chat/completions", "llm/v1/messages"} {
		t.Run(path, func(t *testing.T) {
			resp, body := ExecRequest(t, R{Method: http.MethodPost, Path: path, Body: message(100_000)})
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
			assert.True(t, json.Valid([]byte(body)), "body: %s", body)
		})
	}

	t.Run("too large", func(t *testing.T) {
		resp, body := ExecRequest(t, R{
			Method: http.MethodPost,
			Path:   "llm/v1/chat/completions",
			Body:   message(maxBodySize),
		})
		assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		assert.Contains(t, body, `"type": "request_too_large"`)
	})
}
