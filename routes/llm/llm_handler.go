package llm

import (
	"net/http"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
)

var RouteList = []ex.Route{}

// LLM requests carry whole conversations, so they get a larger body limit than other endpoints.
const maxBodySize = 1 << 20

// holdStreamSlot takes a slot for a streaming response, which can take long. If there's no free slot, it returns a 503
// for the client, in the same error format as other errors here.
func holdStreamSlot(ex *ex.Exchange) *response.Response {
	if ex.HoldLongSlot() {
		return nil
	}
	resp := response.LongBusy()
	resp.Body = map[string]any{
		"error": map[string]any{
			"type":    "overloaded_error",
			"message": resp.Body,
		},
	}
	return &resp
}

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
