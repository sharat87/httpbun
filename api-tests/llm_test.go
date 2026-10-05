package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLLMAnthropicMessagesWithoutV1(t *testing.T) {
	for _, path := range []string{"llm/v1/messages", "llm/messages"} {
		t.Run(path, func(t *testing.T) {
			resp, body := ExecRequest(R{
				Method: http.MethodPost,
				Path:   path,
				Body:   `{"max_tokens": 10, "messages": [{"role": "user", "content": "Hi"}]}`,
			})
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Contains(t, body, `"type": "message"`)
		})
	}
}
