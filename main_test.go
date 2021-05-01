package main

import (
	"strings"
	"net/http/httptest"
	"io"
	"testing"
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/assert"
	"net/http"
	"encoding/json"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(TSuite))
}

type TSuite struct {
	suite.Suite
	Mux http.Handler
}

func (s *TSuite) SetupTest() {
	s.Mux = makeBunHandler()
}

func (s *TSuite) ExecRequest(method, path string, body *string, headers map[string][]string) (http.Response, []byte) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(*body)
	}

	req := httptest.NewRequest(method, "http://example.com/" + path, bodyReader)

	for name, values := range headers {
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

func (s *TSuite) TestMethodGet() {
	resp, body := s.ExecRequest("GET", "get", nil, nil)
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]interface{}{
		"args": make(map[string]interface{}),
		"headers": make(map[string]interface{}),
		"origin": "example.com",
		"url": "http://example.com/get",
	}, parseJson(body))
}

func (s *TSuite) TestMethodGetWithCustomHeaders() {
	resp, body := s.ExecRequest("GET", "get", nil, map[string][]string{
		"X-One": []string{"custom header value"},
		"X-Two": []string{"another custom header"},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]interface{}{
		"args": make(map[string]interface{}),
		"headers": map[string]interface{}{
			"X-One": "custom header value",
			"X-Two": "another custom header",
		},
		"origin": "example.com",
		"url": "http://example.com/get",
	}, parseJson(body))
}

func (s *TSuite) TestMethodGetWithMultipleHeaderValues() {
	resp, body := s.ExecRequest("GET", "get", nil, map[string][]string{
		"X-One": []string{"custom header value", "another custom header"},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]interface{}{
		"args": make(map[string]interface{}),
		"headers": map[string]interface{}{
			"X-One": "custom header value, another custom header",
		},
		"origin": "example.com",
		"url": "http://example.com/get",
	}, parseJson(body))
}

func (s *TSuite) TestMethodPost() {
	resp, body := s.ExecRequest("POST", "post", nil, nil)
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]interface{}{
		"args": make(map[string]interface{}),
		"headers": make(map[string]interface{}),
		"origin": "example.com",
		"url": "http://example.com/post",
	}, parseJson(body))
}

func parseJson(raw []byte) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		panic(err)
	}
	return data
}

func TestComputeDigestAuthResponse(t *testing.T) {
	response := computeDigestAuthResponse(
		"Mufasa",
		"Circle Of Life",
		"dcd98b7102dd2f0e8b11d0f600bfb0c093",
		"00000001",
		"0a4f113b",
		"auth",
		"GET",
		"/dir/index.html",
	)
	assert.Equal(
		t,
		"6629fae49393a05397450978507c4ef1",
		response,
	)
}
