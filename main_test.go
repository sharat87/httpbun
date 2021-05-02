package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(TSuite))
}

type TSuite struct {
	suite.Suite
	Mux http.Handler
}

type R struct {
	Method  string
	Path    string
	Body    string
	Headers map[string][]string
}

func (s *TSuite) SetupTest() {
	s.Mux = makeBunHandler()
}

func (s *TSuite) ExecRequest(request R) (http.Response, []byte) {
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

func (s *TSuite) TestMethodGet() {
	resp, body := s.ExecRequest(R{
		Method: "GET",
		Path:   "get",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]interface{}{
		"args":    make(map[string]interface{}),
		"headers": make(map[string]interface{}),
		"origin":  "example.com",
		"url":     "http://example.com/get",
	}, parseJson(body))
}

func (s *TSuite) TestMethodGetWithCustomHeaders() {
	resp, body := s.ExecRequest(R{
		Method: "GET",
		Path:   "get",
		Headers: map[string][]string{
			"X-One": []string{"custom header value"},
			"X-Two": []string{"another custom header"},
		},
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
		"url":    "http://example.com/get",
	}, parseJson(body))
}

func (s *TSuite) TestMethodGetWithMultipleHeaderValues() {
	resp, body := s.ExecRequest(R{
		Method: "GET",
		Path:   "get",
		Headers: map[string][]string{
			"X-One": []string{"custom header value", "another custom header"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]interface{}{
		"args": make(map[string]interface{}),
		"headers": map[string]interface{}{
			"X-One": "custom header value, another custom header",
		},
		"origin": "example.com",
		"url":    "http://example.com/get",
	}, parseJson(body))
}

func (s *TSuite) TestMethodPost() {
	resp, body := s.ExecRequest(R{
		Method: "POST",
		Path:   "post",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]interface{}{
		"args":    make(map[string]interface{}),
		"form":    make(map[string]interface{}),
		"data":    "",
		"headers": make(map[string]interface{}),
		"origin":  "example.com",
		"url":     "http://example.com/post",
	}, parseJson(body))
}

func (s *TSuite) TestMethodPostWithPlainBody() {
	resp, body := s.ExecRequest(R{
		Method: "POST",
		Path:   "post",
		Body:   "answer=42",
		Headers: map[string][]string{
			"Content-Type": []string{"application/x-www-form-urlencoded"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal(map[string]interface{}{
		"args": make(map[string]interface{}),
		"form": map[string]interface{}{
			"answer": "42",
		},
		"data": "",
		"headers": map[string]interface{}{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		"origin": "example.com",
		"url":    "http://example.com/post",
	}, parseJson(body))
}

func (s *TSuite) TestBasicAuthWithoutCreds() {
	resp, body := s.ExecRequest(R{
		Method: "GET",
		Path:   "basic-auth/scott/tiger",
	})
	s.Equal(401, resp.StatusCode)
	s.Equal("Basic realm=\"Fake Realm\"", resp.Header.Get("WWW-Authenticate"))
	s.Equal(len(body), 0)
}

func (s *TSuite) TestBasicAuthWithValidCreds() {
	resp, body := s.ExecRequest(R{
		Method: "GET",
		Path:   "basic-auth/scott/tiger",
		Headers: map[string][]string{
			"Authorization": []string{"Basic c2NvdHQ6dGlnZXI="},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("", resp.Header.Get("WWW-Authenticate"))
	s.Equal(map[string]interface{}{
		"authenticated": true,
		"user":          "scott",
	}, parseJson(body))
}

func (s *TSuite) TestBasicAuthWithInvalidCreds() {
	resp, body := s.ExecRequest(R{
		Method: "GET",
		Path:   "basic-auth/scott/tiger",
		Headers: map[string][]string{
			"Authorization": []string{"Basic x2NvdHQ6dGlnZXI="},
		},
	})
	s.Equal(401, resp.StatusCode)
	s.Equal("Basic realm=\"Fake Realm\"", resp.Header.Get("WWW-Authenticate"))
	s.Equal(len(body), 0)
}

func (s *TSuite) TestBearerAuthWithoutToken() {
	resp, body := s.ExecRequest(R{
		Method: "GET",
		Path:   "bearer",
	})
	s.Equal(401, resp.StatusCode)
	s.Equal("Bearer", resp.Header.Get("WWW-Authenticate"))
	s.Equal(len(body), 0)
}

func (s *TSuite) TestBearerAuthWithToken() {
	resp, body := s.ExecRequest(R{
		Method: "GET",
		Path:   "bearer",
		Headers: map[string][]string{
			"Authorization": []string{"Bearer my-auth-token"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("", resp.Header.Get("WWW-Authenticate"))
	s.Equal(map[string]interface{}{
		"authenticated": true,
		"token":         "my-auth-token",
	}, parseJson(body))
}

func (s *TSuite) TestDigestAuthWithoutCreds() {
	resp, body := s.ExecRequest(R{
		Method: "GET",
		Path:   "digest-auth/auth/dave/diamond",
	})
	s.Equal(401, resp.StatusCode)
	match := regexp.MustCompile("\\bnonce=(\\S+)").FindStringSubmatch(resp.Header.Get("Set-Cookie"))
	if !s.NotEmpty(match) {
		return
	}
	nonce := match[1]
	m := regexp.MustCompile(
		"Digest realm=\"testrealm@host.com\", qop=\"auth,auth-int\", nonce=\"" + nonce + "\", opaque=\"[a-z0-9]+\", algorithm=MD5, stale=FALSE",
	).FindString(resp.Header.Get("WWW-Authenticate"))
	s.NotEmpty(m, "Unexpected value for WWW-Authenticate")
	s.Equal(len(body), 0)
}

func (s *TSuite) TestDigestAuthWitCreds() {
	resp, body := s.ExecRequest(R{
		Method: "GET",
		Path: "digest-auth/auth/dave/diamond",
		Headers: map[string][]string{
			"Cookie": []string{"nonce=d9fc96d7fe39099441042eea21006d77"},
			"Authorization": []string{"Digest username=\"dave\", realm=\"testrealm@host.com\", nonce=\"d9fc96d7fe39099441042eea21006d77\", uri=\"/digest-auth/auth/dave/diamond\", algorithm=MD5, response=\"10c1132a06ac0de7c39a07e8553f0f14\", opaque=\"362d9b0fe6787b534eb27677f4210b61\", qop=auth, nc=00000001, cnonce=\"bb2ec71d21a27e19\""},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Empty(resp.Header.Get("Set-Cookie"))
	s.Equal(map[string]interface{}{
		"authenticated": true,
		"user":         "dave",
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
