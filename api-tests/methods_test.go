package api_tests

import (
	tu "github.com/sharat87/httpbun/test_utils"
)

func (s *TSuite) TestGet() {
	resp, body := s.ExecRequest(tu.R{
		Method: "GET",
		Path:   "get",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("httpbun//", resp.Header.Get("X-Powered-By"))
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal("185", resp.Header.Get("Content-Length"))
	s.Equal(map[string]any{
		"args": map[string]any{},
		"headers": map[string]any{
			"Accept-Encoding": "gzip",
			"User-Agent":      "httpbun-tests",
		},
		"method": "GET",
		"origin": "127.0.0.1",
		"url":    "http://127.0.0.1:30001/get",
	}, parseJSON(body))
}

func (s *TSuite) TestGetNameSherlock() {
	resp, body := s.ExecRequest(tu.R{
		Method: "GET",
		Path:   "get?name=Sherlock",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal("225", resp.Header.Get("Content-Length"))
	s.Equal(map[string]any{
		"args": map[string]any{
			"name": "Sherlock",
		},
		"headers": map[string]any{
			"Accept-Encoding": "gzip",
			"User-Agent":      "httpbun-tests",
		},
		"method": "GET",
		"origin": "127.0.0.1",
		"url":    "http://127.0.0.1:30001/get?name=Sherlock",
	}, parseJSON(body))
}

func (s *TSuite) TestGetFirstSherlockLastHolmes() {
	resp, body := s.ExecRequest(tu.R{
		Method: "GET",
		Path:   "get?first=Sherlock&last=Holmes",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal("261", resp.Header.Get("Content-Length"))
	s.Equal(map[string]any{
		"args": map[string]any{
			"first": "Sherlock",
			"last":  "Holmes",
		},
		"headers": map[string]any{
			"Accept-Encoding": "gzip",
			"User-Agent":      "httpbun-tests",
		},
		"method": "GET",
		"origin": "127.0.0.1",
		"url":    "http://127.0.0.1:30001/get?first=Sherlock&last=Holmes",
	}, parseJSON(body))
}

func (s *TSuite) TestGetWithCustomHeader() {
	resp, body := s.ExecRequest(tu.R{
		Method: "GET",
		Path:   "get",
		Headers: map[string][]string{
			"X-Custom": {"first-custom"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal("217", resp.Header.Get("Content-Length"))
	s.Equal(map[string]any{
		"args": map[string]any{},
		"headers": map[string]any{
			"Accept-Encoding": "gzip",
			"User-Agent":      "httpbun-tests",
			"X-Custom":        "first-custom",
		},
		"method": "GET",
		"origin": "127.0.0.1",
		"url":    "http://127.0.0.1:30001/get",
	}, parseJSON(body))
}

func (s *TSuite) TestGetWithTwoCustomHeader() {
	resp, body := s.ExecRequest(tu.R{
		Method: "GET",
		Path:   "get",
		Headers: map[string][]string{
			"X-First":  {"first-custom"},
			"X-Second": {"second-custom"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal("249", resp.Header.Get("Content-Length"))
	s.Equal(map[string]any{
		"args": map[string]any{},
		"headers": map[string]any{
			"Accept-Encoding": "gzip",
			"User-Agent":      "httpbun-tests",
			"X-First":         "first-custom",
			"X-Second":        "second-custom",
		},
		"method": "GET",
		"origin": "127.0.0.1",
		"url":    "http://127.0.0.1:30001/get",
	}, parseJSON(body))
}
