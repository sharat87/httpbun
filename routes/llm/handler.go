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
)

var RouteList = []ex.Route{
	ex.NewRoute(`/llm/proxy`, handleProxy),
}

type ProxyRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func handleProxy(ex *ex.Exchange) response.Response {
	if ex.Request.Method != http.MethodPost {
		return response.Response{
			Status: http.StatusMethodNotAllowed,
			Body:   "Only POST is allowed",
		}
	}

	var payload ProxyRequest
	err := json.Unmarshal(ex.BodyBytes(), &payload)
	if err != nil {
		return response.BadRequest("Invalid JSON payload: %v", err)
	}

	if payload.Provider == "" {
		return response.BadRequest("Missing `provider` in payload")
	}

	if payload.Model == "" {
		return response.BadRequest("Missing `model` in payload")
	}

	if len(payload.Messages) == 0 {
		return response.BadRequest("Missing `messages` in payload")
	}

	switch payload.Provider {
	case "fake":
		return handleFakeProvider(payload)
	default:
		return response.BadRequest("Unsupported provider: %s", payload.Provider)
	}
}

func handleFakeProvider(payload ProxyRequest) response.Response {
	return response.Response{
		Status: http.StatusOK,
		Header: http.Header{
			c.ContentType:   {"text/event-stream"},
			"Cache-Control": {"no-cache"},
			"Connection":    {"keep-alive"},
		},
		Writer: func(w response.BodyWriter) {
			for _, message := range payload.Messages {
				// Simulate a delay
				time.Sleep(100 * time.Millisecond)

				// Create a fake response chunk
				chunk := fmt.Sprintf("data: {\"content\": \"%s\"}\n\n", strings.ToUpper(message.Content))
				err := w.Write(chunk)
				if err != nil {
					return // Client disconnected
				}
			}

			// Signal the end of the stream
			_ = w.Write("data: [DONE]\n\n")
		},
	}
}
