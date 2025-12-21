package llm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/util"
)

var RouteList = []ex.Route{
	// OpenAI, default base_url in SDKs is "https://api.openai.com/v1"
	ex.NewRoute("/llm/completions", handleCompletions),
	ex.NewRoute("/llm/chat/completions", handleChatCompletions),
	ex.NewRoute("/llm/responses", handleResponses),
}

// CompletionRequest represents the request body for the completions endpoint
type CompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      any     `json:"prompt"` // string or []string
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	N           int     `json:"n"`
	Stream      bool    `json:"stream"`
	Stop        any     `json:"stop"` // string or []string
	User        string  `json:"user"`
	Suffix      string  `json:"suffix"`
}

// ChatCompletionRequest represents the request body for the chat completions endpoint
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	N           int           `json:"n"`
	Stream      bool          `json:"stream"`
	Stop        any           `json:"stop"`
	User        string        `json:"user"`
	Httpbun     *HttpbunMock  `json:"httpbun,omitempty"`
}

type ResponsesRequest struct {
	Model   string         `json:"model"`
	Input   any            `json:"input"`
	User    string         `json:"user"`
	Stream  bool           `json:"stream"`
	Httpbun *ResponsesMock `json:"httpbun,omitempty"`
}

type ResponsesMock struct {
	OutputText string `json:"output_text"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// HttpbunMock allows customizing the mock response
type HttpbunMock struct {
	Content string `json:"content"`
}

func handleCompletions(ex *ex.Exchange) response.Response {
	if ex.Request.Method != http.MethodPost {
		return response.Response{
			Status: http.StatusMethodNotAllowed,
			Header: http.Header{"Allow": []string{http.MethodPost}},
			Body:   map[string]any{"error": "Method not allowed"},
		}
	}

	var req CompletionRequest
	if err := json.Unmarshal(ex.BodyBytes(), &req); err != nil {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error": map[string]any{
					"message": "Invalid JSON in request body: " + err.Error(),
					"type":    "invalid_request_error",
				},
			},
		}
	}

	if req.Model == "" {
		req.Model = "gpt-3.5-turbo-instruct"
	}
	if req.N == 0 {
		req.N = 1
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 16
	}

	// Get the prompt text for token counting
	promptText := getPromptText(req.Prompt)
	promptTokens := estimateTokens(promptText)

	// Generate mock response text
	mockText := "This is a mock completion response from httpbun. Your prompt was received successfully."

	if req.Stream {
		return streamCompletionResponse(req, mockText, promptTokens)
	}

	completionTokens := estimateTokens(mockText)
	choices := make([]map[string]any, req.N)
	for i := 0; i < req.N; i++ {
		choices[i] = map[string]any{
			"text":          mockText,
			"index":         i,
			"logprobs":      nil,
			"finish_reason": "stop",
		}
	}

	return response.Response{
		Header: http.Header{c.ContentType: []string{c.ApplicationJSON}},
		Body: map[string]any{
			"id":      "cmpl-" + util.RandomString()[:24],
			"object":  "text_completion",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": choices,
			"usage": map[string]any{
				"prompt_tokens":     promptTokens,
				"completion_tokens": completionTokens,
				"total_tokens":      promptTokens + completionTokens,
			},
		},
	}
}

func handleChatCompletions(ex *ex.Exchange) response.Response {
	if ex.Request.Method != http.MethodPost {
		return response.Response{
			Status: http.StatusMethodNotAllowed,
			Header: http.Header{"Allow": []string{http.MethodPost}},
			Body:   map[string]any{"error": "Method not allowed"},
		}
	}

	var req ChatCompletionRequest
	if err := json.Unmarshal(ex.BodyBytes(), &req); err != nil {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error": map[string]any{
					"message": "Invalid JSON in request body: " + err.Error(),
					"type":    "invalid_request_error",
				},
			},
		}
	}

	if req.Model == "" {
		req.Model = "gpt-4"
	}
	if req.N == 0 {
		req.N = 1
	}

	// Count prompt tokens from all messages
	var promptText string
	for _, msg := range req.Messages {
		promptText += msg.Role + ": " + msg.Content + "\n"
	}
	promptTokens := estimateTokens(promptText)

	// Generate mock response - use httpbun.content if provided
	mockContent := "This is a mock chat response from httpbun. I received your messages and I'm responding with this placeholder text."
	if req.Httpbun != nil && req.Httpbun.Content != "" {
		mockContent = req.Httpbun.Content
	}

	if req.Stream {
		return streamChatCompletionResponse(req, mockContent, promptTokens)
	}

	completionTokens := estimateTokens(mockContent)
	choices := make([]map[string]any, req.N)
	for i := 0; i < req.N; i++ {
		choices[i] = map[string]any{
			"index": i,
			"message": map[string]any{
				"role":    "assistant",
				"content": mockContent,
			},
			"finish_reason": "stop",
		}
	}

	return response.Response{
		Header: http.Header{c.ContentType: []string{c.ApplicationJSON}},
		Body: map[string]any{
			"id":      "chatcmpl-" + util.RandomString()[:24],
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": choices,
			"usage": map[string]any{
				"prompt_tokens":     promptTokens,
				"completion_tokens": completionTokens,
				"total_tokens":      promptTokens + completionTokens,
			},
		},
	}
}

func handleResponses(ex *ex.Exchange) response.Response {
	if ex.Request.Method != http.MethodPost {
		return response.Response{
			Status: http.StatusMethodNotAllowed,
			Header: http.Header{"Allow": []string{http.MethodPost}},
			Body:   map[string]any{"error": "Method not allowed"},
		}
	}

	var req ResponsesRequest
	if err := json.Unmarshal(ex.BodyBytes(), &req); err != nil {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error": map[string]any{
					"message": "Invalid JSON in request body: " + err.Error(),
					"type":    "invalid_request_error",
				},
			},
		}
	}

	if req.Model == "" {
		req.Model = "gpt-4.1-mini"
	}

	mockOutputText := "This is a mock responses API response from httpbun. I received your input and I'm responding with this placeholder text."
	if req.Httpbun != nil && req.Httpbun.OutputText != "" {
		mockOutputText = req.Httpbun.OutputText
	}

	inputText := getResponsesInputText(req.Input)
	inputTokens := estimateTokensWithPadding(inputText)
	outputTokens := 29
	totalTokens := inputTokens + outputTokens

	outputMessage := map[string]any{
		"id":     "msg-" + util.RandomString()[:24],
		"type":   "message",
		"role":   "assistant",
		"status": "completed",
		"content": []map[string]any{
			{
				"type":        "output_text",
				"text":        mockOutputText,
				"annotations": []any{},
				"logprobs":    nil,
			},
		},
	}

	usage := map[string]any{
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"total_tokens":  totalTokens,
		"input_tokens_details": map[string]any{
			"cached_tokens": 0,
		},
		"output_tokens_details": map[string]any{
			"reasoning_tokens": 0,
		},
	}

	responseBody := map[string]any{
		"id":                  "resp-" + util.RandomString()[:24],
		"object":              "response",
		"created_at":          float64(time.Now().Unix()),
		"model":               req.Model,
		"status":              "completed",
		"error":               nil,
		"output":              []any{outputMessage},
		"output_text":         mockOutputText,
		"usage":               usage,
		"parallel_tool_calls": false,
		"tool_choice":         "auto",
		"tools":               []any{},
	}

	return response.Response{
		Header: http.Header{c.ContentType: []string{c.ApplicationJSON}},
		Body:   responseBody,
	}
}

func streamCompletionResponse(req CompletionRequest, mockText string, promptTokens int) response.Response {
	return response.Response{
		Header: http.Header{
			c.ContentType:   []string{"text/event-stream"},
			"Cache-Control": []string{"no-cache"},
		},
		Writer: func(w response.BodyWriter) {
			words := strings.Fields(mockText)
			completionID := "cmpl-" + util.RandomString()[:24]

			for i, word := range words {
				text := word
				if i < len(words)-1 {
					text += " "
				}

				chunk := map[string]any{
					"id":      completionID,
					"object":  "text_completion",
					"created": time.Now().Unix(),
					"model":   req.Model,
					"choices": []map[string]any{
						{
							"text":          text,
							"index":         0,
							"logprobs":      nil,
							"finish_reason": nil,
						},
					},
				}

				data, _ := json.Marshal(chunk)
				w.Write("data: " + string(data) + "\n\n")
				time.Sleep(50 * time.Millisecond)
			}

			// Send final chunk with finish_reason
			finalChunk := map[string]any{
				"id":      completionID,
				"object":  "text_completion",
				"created": time.Now().Unix(),
				"model":   req.Model,
				"choices": []map[string]any{
					{
						"text":          "",
						"index":         0,
						"logprobs":      nil,
						"finish_reason": "stop",
					},
				},
			}
			data, _ := json.Marshal(finalChunk)
			w.Write("data: " + string(data) + "\n\n")
			w.Write("data: [DONE]\n\n")
		},
	}
}

func streamChatCompletionResponse(req ChatCompletionRequest, mockContent string, promptTokens int) response.Response {
	return response.Response{
		Header: http.Header{
			c.ContentType:   []string{"text/event-stream"},
			"Cache-Control": []string{"no-cache"},
		},
		Writer: func(w response.BodyWriter) {
			words := strings.Fields(mockContent)
			completionID := "chatcmpl-" + util.RandomString()[:24]

			// Send initial chunk with role
			initialChunk := map[string]any{
				"id":      completionID,
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   req.Model,
				"choices": []map[string]any{
					{
						"index": 0,
						"delta": map[string]any{
							"role": "assistant",
						},
						"finish_reason": nil,
					},
				},
			}
			data, _ := json.Marshal(initialChunk)
			w.Write("data: " + string(data) + "\n\n")
			time.Sleep(50 * time.Millisecond)

			// Stream content word by word
			for i, word := range words {
				content := word
				if i < len(words)-1 {
					content += " "
				}

				chunk := map[string]any{
					"id":      completionID,
					"object":  "chat.completion.chunk",
					"created": time.Now().Unix(),
					"model":   req.Model,
					"choices": []map[string]any{
						{
							"index": 0,
							"delta": map[string]any{
								"content": content,
							},
							"finish_reason": nil,
						},
					},
				}

				data, _ := json.Marshal(chunk)
				w.Write("data: " + string(data) + "\n\n")
				time.Sleep(50 * time.Millisecond)
			}

			// Send final chunk with finish_reason
			finalChunk := map[string]any{
				"id":      completionID,
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   req.Model,
				"choices": []map[string]any{
					{
						"index":         0,
						"delta":         map[string]any{},
						"finish_reason": "stop",
					},
				},
			}
			data, _ = json.Marshal(finalChunk)
			w.Write("data: " + string(data) + "\n\n")
			w.Write("data: [DONE]\n\n")
		},
	}
}

func getResponsesInputText(input any) string {
	if input == nil {
		return ""
	}

	switch value := input.(type) {
	case string:
		return value
	case []any:
		var parts []string
		for _, item := range value {
			text := getResponsesInputText(item)
			if text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	case map[string]any:
		if content, ok := value["content"]; ok {
			contentText := getResponsesInputText(content)
			if contentText != "" {
				if role, ok := value["role"].(string); ok && role != "" {
					return role + ": " + contentText
				}
				return contentText
			}
		}
		if text, ok := value["text"].(string); ok {
			return text
		}
		if raw, err := json.Marshal(value); err == nil {
			return string(raw)
		}
	default:
		if raw, err := json.Marshal(value); err == nil {
			return string(raw)
		}
	}

	return fmt.Sprintf("%v", input)
}

func estimateTokensWithPadding(text string) int {
	tokens := estimateTokens(text)
	if tokens == 0 {
		return 0
	}
	return tokens + 1
}

// getPromptText extracts the prompt text from the request (handles string or []string)
func getPromptText(prompt any) string {
	if prompt == nil {
		return ""
	}
	switch p := prompt.(type) {
	case string:
		return p
	case []any:
		var parts []string
		for _, item := range p {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	}
	return fmt.Sprintf("%v", prompt)
}

// estimateTokens provides a rough estimate of token count (approx 4 chars per token)
func estimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4
}
