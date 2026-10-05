package llm

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/ex"
)

func TestSplitIntoChunks(t *testing.T) {
	for _, text := range []string{"", "one", "one two", "  lead and trail  ", "line1\n\nline2  x", "tab\there"} {
		chunks := splitIntoChunks(text)
		assert.Equal(t, text, strings.Join(chunks, ""), "chunks: %q", chunks)
	}
	assert.Equal(t, []string{"Hello\n\n", "World"}, splitIntoChunks("Hello\n\nWorld"))
}

func post(path, body string, handler ex.HandlerFn) map[string]any {
	resp := ex.InvokeHandlerForTest(path, http.Request{
		Method: http.MethodPost,
		Body:   io.NopCloser(bytes.NewBufferString(body)),
	}, "/"+path, handler)
	result, _ := resp.Body.(map[string]any)
	return result
}

func TestChatCompletionsWithContentParts(t *testing.T) {
	body := post("llm/v1/chat/completions",
		`{"messages": [{"role": "user", "content": [{"type": "text", "text": "Hi there"}]}]}`,
		handleChatCompletions)
	usage, ok := body["usage"].(map[string]any)
	if assert.True(t, ok, "body: %v", body) {
		assert.Equal(t, 4, usage["prompt_tokens"]) // "user: Hi there\n"
	}
}

func TestCompletionsWithCustomContent(t *testing.T) {
	body := post("llm/v1/completions", `{"prompt": "hi", "httpbun": {"content": "custom"}}`, handleCompletions)
	choices := body["choices"].([]map[string]any)
	assert.Equal(t, "custom", choices[0]["text"])
}

func postStatus(path, body string, handler ex.HandlerFn) int {
	resp := ex.InvokeHandlerForTest(path, http.Request{
		Method: http.MethodPost,
		Body:   io.NopCloser(bytes.NewBufferString(body)),
	}, "/"+path, handler)
	if resp.Status == 0 {
		return http.StatusOK
	}
	return resp.Status
}

func TestLargeRequestBody(t *testing.T) {
	message := func(size int) string {
		return `{"messages": [{"role": "user", "content": "` + strings.Repeat("a", size) + `"}]}`
	}
	// Larger than the 10KB limit for other endpoints.
	assert.Equal(t, http.StatusOK, postStatus("llm/v1/chat/completions", message(100_000), handleChatCompletions))
	assert.Equal(t, http.StatusOK, postStatus("llm/v1/messages", message(100_000), handleMessages))
	assert.Equal(t, http.StatusRequestEntityTooLarge, postStatus("llm/v1/chat/completions", message(maxBodySize), handleChatCompletions))
}
