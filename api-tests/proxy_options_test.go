package api_tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/server/spec"
)

// proxySpec is configured like httpbun behind a typical platform proxy.
var proxySpec = spec.Spec{
	ClientIPHeader: "X-Real-Ip",
	SchemeHeader:   "X-Forwarded-Proto",
	HideHeaders:    []string{"X-Forwarded-*", "X-Request-Start"},
}

func echoedHeaders(t *testing.T, body string) map[string]any {
	t.Helper()
	var data struct{ Headers map[string]any }
	assert.NoError(t, json.Unmarshal([]byte(body), &data))
	return data.Headers
}

func TestClientIPHeaderDefaultIgnoresOtherHeaders(t *testing.T) {
	s := assert.New(t)
	_, body := ExecRequest(t, R{Path: "ip", Headers: map[string][]string{"X-Real-Ip": {"12.34.56.78"}}})
	s.NotContains(body, "12.34.56.78", "only X-Httpbun-Forwarded-For is trusted by default")
}

func TestClientIPHeader(t *testing.T) {
	s := assert.New(t)
	srv := NewServer(t, proxySpec)
	_, body := srv.Exec(t, R{Path: "ip", Headers: map[string][]string{
		"X-Real-Ip":               {"12.34.56.78"},
		"X-Httpbun-Forwarded-For": {"98.76.54.32"},
	}})
	s.JSONEq(`{"origin": "12.34.56.78"}`, body)
}

func TestSchemeHeader(t *testing.T) {
	s := assert.New(t)
	srv := NewServer(t, proxySpec)
	_, body := srv.Exec(t, R{Path: "get", Headers: map[string][]string{"X-Forwarded-Proto": {"https"}}})
	s.Contains(body, `"url": "https://`+Host+`/get"`)
}

func TestSchemeHeaderDefaultIgnoresOtherHeaders(t *testing.T) {
	s := assert.New(t)
	_, body := ExecRequest(t, R{Path: "get", Headers: map[string][]string{"X-Forwarded-Proto": {"https"}}})
	s.Contains(body, `"url": "http://`+Host+`/get"`)
}

func TestHideHeaders(t *testing.T) {
	s := assert.New(t)
	srv := NewServer(t, proxySpec)
	_, body := srv.Exec(t, R{Path: "headers", Headers: map[string][]string{
		"X-Real-Ip":         {"12.34.56.78"},
		"X-Forwarded-Proto": {"https"},
		"X-Forwarded-For":   {"12.34.56.78, 10.0.0.1"},
		"X-Forwarded-Host":  {"example.com"},
		"X-Request-Start":   {"1234"},
		"X-Custom":          {"kept"},
	}})
	headers := echoedHeaders(t, body)
	for _, name := range []string{"X-Real-Ip", "X-Forwarded-Proto", "X-Forwarded-For", "X-Forwarded-Host", "X-Request-Start"} {
		s.NotContains(headers, name)
	}
	s.Equal("kept", headers["X-Custom"])
}

func TestHideHeadersDefault(t *testing.T) {
	s := assert.New(t)
	_, body := ExecRequest(t, R{Path: "headers", Headers: map[string][]string{
		"X-Forwarded-For":         {"12.34.56.78"},
		"X-Httpbun-Forwarded-For": {"12.34.56.78"},
	}})
	headers := echoedHeaders(t, body)
	s.Equal("12.34.56.78", headers["X-Forwarded-For"])
	s.NotContains(headers, "X-Httpbun-Forwarded-For")
}

func TestAnyHosts(t *testing.T) {
	s := assert.New(t)
	srv := NewServer(t, spec.Spec{AnyHosts: []string{Host}})
	resp, body := srv.Exec(t, R{Path: "some/random/path"})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Contains(body, `"url": "http://`+Host+`/some/random/path"`)

	srv = NewServer(t, spec.Spec{AnyHosts: []string{"any.example.com"}})
	resp, _ = srv.Exec(t, R{Path: "some/random/path"})
	s.Equal(http.StatusNotFound, resp.StatusCode, "other hosts route normally")
}

func TestPageSecurityHeaders(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(t, R{Path: ""})
	s.Equal("nosniff", resp.Header.Get("X-Content-Type-Options"))
	s.Equal("frame-ancestors 'none'", resp.Header.Get("Content-Security-Policy"))
	s.Empty(resp.Header.Get("Strict-Transport-Security"), "HSTS is off by default")

	resp, _ = ExecRequest(t, R{Path: "get"})
	s.Empty(resp.Header.Get("Content-Security-Policy"), "only pages get these")
}

func TestHSTS(t *testing.T) {
	s := assert.New(t)
	srv := NewServer(t, spec.Spec{HSTS: true, SchemeHeader: "X-Forwarded-Proto"})

	resp, _ := srv.Exec(t, R{Path: "", Headers: map[string][]string{"X-Forwarded-Proto": {"https"}}})
	s.Equal("max-age=31536000; includeSubDomains; preload", resp.Header.Get("Strict-Transport-Security"))

	resp, _ = srv.Exec(t, R{Path: ""})
	s.Empty(resp.Header.Get("Strict-Transport-Security"), "not over http")

	resp, _ = srv.Exec(t, R{Path: "get", Headers: map[string][]string{"X-Forwarded-Proto": {"https"}}})
	s.Empty(resp.Header.Get("Strict-Transport-Security"), "only on pages")
}
