package bun

import (
	tu "github.com/sharat87/httpbun/test_utils"
	"github.com/stretchr/testify/suite"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMethodSuite(t *testing.T) {
	suite.Run(t, new(MethodSuite))
}

type MethodSuite struct {
	suite.Suite
	Mux http.Handler
}

func (s *MethodSuite) SetupSuite() {
	s.Mux = MakeBunHandler("", "", "")
}

func (s *MethodSuite) ExecRequest(request tu.R) (http.Response, []byte) {
	var bodyReader io.Reader
	if request.Body != "" {
		bodyReader = strings.NewReader(request.Body)
	}

	req := httptest.NewRequest(request.Method, "http://example.com/"+request.Path, bodyReader)

	for name, values := range request.Headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	resp := w.Result()
	responseBody, _ := io.ReadAll(resp.Body)

	return *resp, responseBody
}

func (s *MethodSuite) TestMethodGet() {
	resp, body := s.ExecRequest(tu.R{
		Method: "GET",
		Path:   "get",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]any{
		"method":  "GET",
		"args":    map[string]any{},
		"headers": map[string]any{},
		"origin":  "192.0.2.1",
		"url":     "http://example.com/get",
	}, tu.ParseJson(body))
}

func (s *MethodSuite) TestMethodGetWithCustomHeaders() {
	resp, body := s.ExecRequest(tu.R{
		Method: "GET",
		Path:   "get",
		Headers: map[string][]string{
			"X-One": {"custom header value"},
			"X-Two": {"another custom header"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]any{
		"method": "GET",
		"args":   map[string]any{},
		"headers": map[string]any{
			"X-One": "custom header value",
			"X-Two": "another custom header",
		},
		"origin": "192.0.2.1",
		"url":    "http://example.com/get",
	}, tu.ParseJson(body))
}

func (s *MethodSuite) TestMethodGetWithMultipleHeaderValues() {
	resp, body := s.ExecRequest(tu.R{
		Method: "GET",
		Path:   "get",
		Headers: map[string][]string{
			"X-One": {"custom header value", "another custom header"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]any{
		"method": "GET",
		"args":   map[string]any{},
		"headers": map[string]any{
			"X-One": "custom header value,another custom header",
		},
		"origin": "192.0.2.1",
		"url":    "http://example.com/get",
	}, tu.ParseJson(body))
}

func (s *MethodSuite) TestMethodPost() {
	resp, body := s.ExecRequest(tu.R{
		Method: "POST",
		Path:   "post",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]any{
		"method":  "POST",
		"args":    map[string]any{},
		"form":    map[string]any{},
		"data":    "",
		"headers": map[string]any{},
		"json":    nil,
		"files":   map[string]any{},
		"origin":  "192.0.2.1",
		"url":     "http://example.com/post",
	}, tu.ParseJson(body))
}

func (s *MethodSuite) TestMethodPostWithPlainBody() {
	resp, body := s.ExecRequest(tu.R{
		Method: "POST",
		Path:   "post",
		Body:   "answer=42",
		Headers: map[string][]string{
			"Content-Type": {"application/x-www-form-urlencoded"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]any{
		"method": "POST",
		"args":   map[string]any{},
		"form": map[string]any{
			"answer": "42",
		},
		"data": "",
		"headers": map[string]any{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		"json":   nil,
		"files":  map[string]any{},
		"origin": "192.0.2.1",
		"url":    "http://example.com/post",
	}, tu.ParseJson(body))
}
