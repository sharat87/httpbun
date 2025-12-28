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

func init() {
	RouteList = append(RouteList,
		// https://console.anthropic.com/docs/en/api/messages/create
		ex.NewRoute("/llm/v1/messages", handleMessages),
	)
}

// MessagesRequest represents the request body for the Anthropic messages endpoint
type MessagesRequest struct {
	Model         string             `json:"model"`
	Messages      []AnthropicMessage `json:"messages"`
	MaxTokens     int                `json:"max_tokens"`
	Stream        bool               `json:"stream"`
	Temperature   float64            `json:"temperature"`
	TopP          float64            `json:"top_p"`
	TopK          int                `json:"top_k"`
	StopSequences []string           `json:"stop_sequences"`
	Httpbun       *HttpbunMock       `json:"httpbun,omitempty"`
}

// AnthropicMessage represents a message in the Anthropic API format
type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or array of content blocks
}

func handleMessages(ex *ex.Exchange) response.Response {
	if ex.Request.Method != http.MethodPost {
		return response.Response{
			Status: http.StatusMethodNotAllowed,
			Header: http.Header{"Allow": []string{http.MethodPost}},
			Body:   map[string]any{"error": map[string]any{"type": "method_not_allowed", "message": "Method not allowed"}},
		}
	}

	var req MessagesRequest
	if err := json.Unmarshal(ex.BodyBytes(), &req); err != nil {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error": map[string]any{
					"type":    "invalid_request_error",
					"message": "Invalid JSON in request body: " + err.Error(),
				},
			},
		}
	}

	if req.Model == "" {
		req.Model = "claude-3-5-sonnet-20241022"
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 1024
	}

	// Extract text from messages for token counting
	var promptText string
	for _, msg := range req.Messages {
		promptText += msg.Role + ": "
		contentText := getAnthropicMessageContent(msg.Content)
		promptText += contentText + "\n"
	}
	inputTokens := estimateTokens(promptText)

	// Generate mock response - use httpbun.content if provided
	mockContent := "This is a mock Anthropic messages API response from httpbun. I received your messages and I'm responding with this placeholder text."
	if req.Httpbun != nil && req.Httpbun.Content != "" {
		mockContent = req.Httpbun.Content
	}

	if req.Stream {
		return streamMessagesResponse(req, mockContent, inputTokens)
	}

	outputTokens := estimateTokens(mockContent)

	// Build content blocks (Anthropic uses array of content blocks)
	contentBlocks := []map[string]any{
		{
			"type":      "text",
			"text":      mockContent,
			"citations": nil,
		},
	}

	responseBody := map[string]any{
		"id":            "msg-" + util.RandomString()[:24],
		"type":          "message",
		"role":          "assistant",
		"content":       contentBlocks,
		"model":         req.Model,
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		},
	}

	return response.Response{
		Header: http.Header{c.ContentType: []string{c.ApplicationJSON}},
		Body:   responseBody,
	}
}

func streamMessagesResponse(req MessagesRequest, mockContent string, inputTokens int) response.Response {
	return response.Response{
		Header: http.Header{
			c.ContentType:   []string{"text/event-stream"},
			"Cache-Control": []string{"no-cache"},
		},
		Writer: func(w response.BodyWriter) {
			words := strings.Fields(mockContent)
			messageID := "msg-" + util.RandomString()[:24]

			// Send initial message_start event
			messageStart := map[string]any{
				"type": "message_start",
				"message": map[string]any{
					"id":            messageID,
					"type":          "message",
					"role":          "assistant",
					"content":       []any{},
					"model":         req.Model,
					"stop_reason":   nil,
					"stop_sequence": nil,
					"usage": map[string]any{
						"input_tokens":  inputTokens,
						"output_tokens": 0,
					},
				},
			}
			data, _ := json.Marshal(messageStart)
			w.Write("event: message_start\ndata: " + string(data) + "\n\n")
			time.Sleep(50 * time.Millisecond)

			// Send content_block_start
			contentBlockStart := map[string]any{
				"type":          "content_block_start",
				"index":         0,
				"content_block": map[string]any{"type": "text", "text": ""},
			}
			data, _ = json.Marshal(contentBlockStart)
			w.Write("event: content_block_start\ndata: " + string(data) + "\n\n")
			time.Sleep(50 * time.Millisecond)

			// Stream content word by word
			for i, word := range words {
				text := word
				if i < len(words)-1 {
					text += " "
				}

				contentBlockDelta := map[string]any{
					"type":  "content_block_delta",
					"index": 0,
					"delta": map[string]any{
						"type": "text_delta",
						"text": text,
					},
				}
				data, _ = json.Marshal(contentBlockDelta)
				w.Write("event: content_block_delta\ndata: " + string(data) + "\n\n")
				time.Sleep(50 * time.Millisecond)
			}

			outputTokens := estimateTokens(mockContent)

			// Send content_block_stop
			contentBlockStop := map[string]any{
				"type":  "content_block_stop",
				"index": 0,
			}
			data, _ = json.Marshal(contentBlockStop)
			w.Write("event: content_block_stop\ndata: " + string(data) + "\n\n")
			time.Sleep(50 * time.Millisecond)

			// Send message_delta with stop_reason
			messageDelta := map[string]any{
				"type": "message_delta",
				"delta": map[string]any{
					"stop_reason":   "end_turn",
					"stop_sequence": nil,
				},
				"usage": map[string]any{
					"output_tokens": outputTokens,
				},
			}
			data, _ = json.Marshal(messageDelta)
			w.Write("event: message_delta\ndata: " + string(data) + "\n\n")
			time.Sleep(50 * time.Millisecond)

			// Send message_stop
			messageStop := map[string]any{
				"type": "message_stop",
			}
			data, _ = json.Marshal(messageStop)
			w.Write("event: message_stop\ndata: " + string(data) + "\n\n")
		},
	}
}

func getAnthropicMessageContent(content interface{}) string {
	if content == nil {
		return ""
	}

	switch c := content.(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, item := range c {
			if block, ok := item.(map[string]any); ok {
				if textType, ok := block["type"].(string); ok && textType == "text" {
					if text, ok := block["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			} else if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprintf("%v", content)
	}
}
