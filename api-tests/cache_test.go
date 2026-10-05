package api_tests

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/server/spec"
)

func TestEtagConditionalRequests(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		match  bool
	}{
		{"no validator", nil, false},
		{"quoted", []string{`"foo"`}, true},
		{"weak", []string{`W/"foo"`}, true},
		{"list", []string{`"other", "foo"`}, true},
		{"list weak", []string{`"other", W/"foo"`}, true},
		{"repeated first matches", []string{`"foo"`, `"other"`}, true},
		{"repeated last matches", []string{`"other"`, `"foo"`}, true},
		{"whitespace", []string{" \tW/\"foo\"\t, \"other\" "}, true},
		{"empty list entries", []string{`,, "", W/"foo",,`}, true},
		{"wildcard", []string{"*"}, true},
		{"legacy bare", []string{"foo"}, true},
		{"different", []string{`"other"`}, false},
		{"different weak", []string{`W/"other"`}, false},
		{"unterminated", []string{`"foo`}, false},
		{"invalid weak prefix", []string{`w/"foo"`}, false},
		{"invalid suffix", []string{`"foo"extra`}, false},
		{"mixed wildcard", []string{`*, "foo"`}, false},
		{"bare in list", []string{`"other", foo`}, false},
		{"bare repeated with quoted", []string{"foo", `"other"`}, false},
		{"matching prefix invalid suffix", []string{`"foo", "unterminated`}, false},
		{"matching prefix wildcard", []string{`"foo", *`}, false},
		{"tab inside quoted", []string{"\"fo\to\""}, false},
		{"quoted wildcard", []string{`"*"`}, false},
		{"empty", []string{""}, false},
	}
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut} {
		for _, tt := range tests {
			t.Run(method+"/"+tt.name, func(t *testing.T) {
				s := assert.New(t)
				headers := http.Header{}
				if tt.values != nil {
					headers["If-None-Match"] = tt.values
				}
				resp, body := ExecRequest(t, R{
					Method:  method,
					Path:    "etag/foo?one=two",
					Body:    "request body",
					Headers: headers,
				})

				s.Equal(`"foo"`, resp.Header.Get("ETag"))

				if tt.match {
					wantStatus := http.StatusPreconditionFailed
					if method == http.MethodGet || method == http.MethodHead {
						wantStatus = http.StatusNotModified
					}
					s.Equal(wantStatus, resp.StatusCode)
					s.Empty(body)
					return
				}

				s.Equal(http.StatusOK, resp.StatusCode)
				s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
				if method == http.MethodHead {
					s.Empty(body)
					return
				}
				var info struct {
					Method string
					Args   map[string]string
					URL    string
					Data   string
				}
				s.NoError(json.Unmarshal([]byte(body), &info))
				s.Equal(method, info.Method)
				s.Equal(map[string]string{"one": "two"}, info.Args)
				s.Equal(BaseURL+"etag/foo?one=two", info.URL)
				s.Equal("request body", info.Data)
			})
		}
	}
}

func TestEtagOpaqueValues(t *testing.T) {
	for _, tt := range []struct {
		path   string
		opaque string
	}{
		{"foo,bar", "foo,bar"},
		{`foo%5Cbar`, `foo\bar`},
		{"foo+bar", "foo+bar"},
		{"foo%2Bbar", "foo+bar"},
		{"foo%2bbar", "foo+bar"},
		{"foo%252Bbar", "foo%2Bbar"},
		{"foo%25bar", "foo%bar"},
		{"foo%2Fbar", "foo/bar"},
		{"*", "*"},
		{"caf%C3%A9", "café"},
		{"%80", string([]byte{0x80})},
	} {
		t.Run(tt.path, func(t *testing.T) {
			tag := `"` + tt.opaque + `"`

			t.Run("unconditional", func(t *testing.T) {
				s := assert.New(t)
				resp, body := ExecRequest(t, R{Path: "etag/" + tt.path})
				s.Equal(http.StatusOK, resp.StatusCode)
				s.Equal(tag, resp.Header.Get("ETag"))
				s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
				s.NotEmpty(body)
			})

			t.Run("conditional", func(t *testing.T) {
				s := assert.New(t)
				resp, body := ExecRequest(t, R{
					Path:    "etag/" + tt.path,
					Headers: map[string][]string{"If-None-Match": {`"other", W/` + tag}},
				})
				s.Equal(http.StatusNotModified, resp.StatusCode)
				s.Equal(tag, resp.Header.Get("ETag"))
				s.Empty(body)
			})
		})
	}
}

func TestEtagRejectsInvalidOpaqueValues(t *testing.T) {
	for _, opaque := range []string{"foo bar", `foo"bar`, "foo\tbar", "foo\nbar", "foo\x00bar", "foo\x7fbar"} {
		t.Run(url.PathEscape(opaque), func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{Path: "etag/" + url.PathEscape(opaque)})
			s.Equal(http.StatusBadRequest, resp.StatusCode)
			s.Empty(resp.Header.Values("ETag"))
			s.Contains(body, "Invalid ETag value")
		})
	}
}

func TestEtagPathPrefix(t *testing.T) {
	srv := NewServer(t, spec.Spec{PathPrefix: "/mount"})
	for _, path := range []string{"foo+bar", "foo%2Bbar"} {
		t.Run(path, func(t *testing.T) {
			s := assert.New(t)
			resp, body := srv.Exec(t, R{
				Path:    "mount/etag/" + path,
				Headers: map[string][]string{"If-None-Match": {`"foo+bar"`}},
			})
			s.Equal(http.StatusNotModified, resp.StatusCode)
			s.Equal(`"foo+bar"`, resp.Header.Get("ETag"))
			s.Empty(body)
		})
	}
}

func TestCacheConditionalRequests(t *testing.T) {
	for _, tt := range []struct {
		method string
		header string
		want   int
	}{
		{http.MethodGet, "", http.StatusOK},
		{http.MethodGet, "If-None-Match", http.StatusNotModified},
		{http.MethodGet, "If-Modified-Since", http.StatusNotModified},
		{http.MethodHead, "If-None-Match", http.StatusNotModified},
		{http.MethodPost, "If-None-Match", http.StatusPreconditionFailed},
	} {
		t.Run(tt.method+"/"+tt.header, func(t *testing.T) {
			headers := http.Header{}
			if tt.header != "" {
				headers.Set(tt.header, `"x"`)
			}
			resp, _ := ExecRequest(t, R{Method: tt.method, Path: "cache", Headers: headers})
			assert.Equal(t, tt.want, resp.StatusCode)
		})
	}
}
