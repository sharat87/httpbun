package api_tests

import (
	"github.com/sharat87/httpbun/c"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestPayloadGetWithFormBody(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodGet,
		Path:   "payload",
		Body:   "payload for get isn't an abomination",
		Headers: map[string][]string{
			c.ContentType: {"text/a-crazy-content-type"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("text/a-crazy-content-type", resp.Header.Get(c.ContentType))
	s.Equal("payload for get isn't an abomination", body)
}

func TestPayloadPostWithFormBody(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodPost,
		Path:   "payload",
		Body:   "answer=42",
		Headers: map[string][]string{
			c.ContentType: {"application/x-www-form-urlencoded"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("application/x-www-form-urlencoded", resp.Header.Get(c.ContentType))
	s.Equal("answer=42", body)
}

func TestPayloadPutWithJSONBody(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Method: http.MethodPut,
		Path:   "payload",
		Body:   "true",
		Headers: map[string][]string{
			c.ContentType: {c.ApplicationJSON},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.Equal("true", body)
}
