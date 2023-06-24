package api_tests

import (
	"encoding/json"
	tu "github.com/sharat87/httpbun/test_utils"
	"github.com/stretchr/testify/suite"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(TSuite))
}

type TSuite struct {
	suite.Suite
}

func (s *TSuite) ExecRequest(r tu.R) (http.Response, []byte) {
	var bodyReader io.Reader
	if r.Body != "" {
		bodyReader = strings.NewReader(r.Body)
	}

	//goland:noinspection HttpUrlsUsage
	req, err := http.NewRequest(r.Method, "http://"+os.Getenv("HTTPBUN_BIND")+"/"+r.Path, bodyReader)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "httpbun-tests")
	for name, values := range r.Headers {
		req.Header[name] = values
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(resp.Body)

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return *resp, bodyText
}

func parseJSON(body []byte) map[string]any {
	parsedBody := map[string]any{}
	if err := json.Unmarshal(body, &parsedBody); err != nil {
		log.Fatal(err)
	}
	return parsedBody
}

func (s *TSuite) TestGet() {
	resp, body := s.ExecRequest(tu.R{
		Method: "GET",
		Path:   "get",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("httpbun", resp.Header.Get("X-Powered-By"))
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
		"url":    "http://localhost:30001/get",
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
		"url":    "http://localhost:30001/get?name=Sherlock",
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
		"url":    "http://localhost:30001/get?first=Sherlock&last=Holmes",
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
		"url":    "http://localhost:30001/get",
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
		"url":    "http://localhost:30001/get",
	}, parseJSON(body))
}
