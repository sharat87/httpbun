package llm

import (
	"net/http"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
)

var RouteList = []ex.Route{}

// LLM requests carry whole conversations, so they get a larger body limit than other endpoints.
const maxBodySize = 1 << 20

// readBody reads the request body, or returns an error response if it's too large.
func readBody(ex *ex.Exchange) ([]byte, *response.Response) {
	body, err := ex.BodyBytesWithLimit(maxBodySize)
	if err != nil {
		return nil, &response.Response{
			Status: http.StatusRequestEntityTooLarge,
			Body: map[string]any{
				"error": map[string]any{
					"type":    "request_too_large",
					"message": err.Error(),
				},
			},
		}
	}
	return body, nil
}
