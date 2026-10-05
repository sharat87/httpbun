package api_tests

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/sharat87/httpbun/c"
)

func TestEtagConditionalRequests(t *testing.T) {
	validators := []struct {
		name   string
		values []string
	}{
		{"quoted", []string{`"foo"`}},
		{"weak", []string{`W/"foo"`}},
		{"list", []string{`"other", "foo"`}},
		{"empty list entries", []string{`,, "", W/"foo",,`}},
		{"repeated", []string{`"foo"`, `"other"`}},
		{"wildcard", []string{"*"}},
		{"legacy bare", []string{"foo"}},
	}
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut} {
		for _, validator := range validators {
			t.Run(method+"/"+validator.name, func(t *testing.T) {
				resp, body := ExecRequest(t, R{
					Method: method,
					Path:   "etag/foo",
					Body:   "request body",
					Headers: map[string][]string{
						"If-None-Match": validator.values,
					},
				})
				wantStatus := http.StatusPreconditionFailed
				if method == http.MethodGet || method == http.MethodHead {
					wantStatus = http.StatusNotModified
				}
				if resp.StatusCode != wantStatus {
					t.Fatalf("status = %d, want %d", resp.StatusCode, wantStatus)
				}
				if got := resp.Header.Get("ETag"); got != `"foo"` {
					t.Fatalf("ETag = %q, want %q", got, `"foo"`)
				}
				if body != "" {
					t.Fatalf("body = %q, want empty", body)
				}
			})
		}
	}
}

func TestEtagWithoutMatchingValidator(t *testing.T) {
	for _, values := range [][]string{nil, {`"other"`}, {`"foo`}, {`"foo"extra`}, {`"foo", "unterminated`}, {`"foo", *`}, {"foo", `"other"`}, {"\"fo\to\""}} {
		t.Run(http.Header{"If-None-Match": values}.Get("If-None-Match"), func(t *testing.T) {
			resp, body := ExecRequest(t, R{
				Path:    "etag/foo?one=two",
				Headers: map[string][]string{"If-None-Match": values},
			})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
			if got := resp.Header.Get("ETag"); got != `"foo"` {
				t.Fatalf("ETag = %q, want %q", got, `"foo"`)
			}
			if got := resp.Header.Get(c.ContentType); got != c.ApplicationJSON {
				t.Fatalf("Content-Type = %q, want %q", got, c.ApplicationJSON)
			}
			var info struct {
				Method string
				Args   map[string]string
				URL    string
			}
			if err := json.Unmarshal([]byte(body), &info); err != nil {
				t.Fatal(err)
			}
			if info.Method != http.MethodGet || info.Args["one"] != "two" || info.URL != BaseURL+"etag/foo?one=two" {
				t.Fatalf("body = %q, want request information", body)
			}
		})
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
	} {
		t.Run(tt.path, func(t *testing.T) {
			tag := `"` + tt.opaque + `"`
			for _, conditional := range []bool{false, true} {
				headers := http.Header{}
				wantStatus := http.StatusOK
				if conditional {
					headers.Set("If-None-Match", `"other", W/`+tag)
					wantStatus = http.StatusNotModified
				}
				resp, body := ExecRequest(t, R{
					Path:    "etag/" + tt.path,
					Headers: headers,
				})
				if resp.StatusCode != wantStatus {
					t.Fatalf("conditional = %t: status = %d, want %d", conditional, resp.StatusCode, wantStatus)
				}
				if got := resp.Header.Get("ETag"); got != tag {
					t.Fatalf("ETag = %q, want %q", got, tag)
				}
				if conditional && body != "" {
					t.Fatalf("body = %q, want empty", body)
				}
			}
		})
	}
}

func TestEtagRejectsInvalidOpaqueValues(t *testing.T) {
	for _, opaque := range []string{"foo bar", `foo"bar`, "foo\tbar", "foo\x7fbar"} {
		t.Run(url.PathEscape(opaque), func(t *testing.T) {
			resp, _ := ExecRequest(t, R{Path: "etag/" + url.PathEscape(opaque)})
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", resp.StatusCode)
			}
			if got := resp.Header.Get("ETag"); got != "" {
				t.Fatalf("invalid opaque value emitted ETag %q", got)
			}
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
			if resp.StatusCode != tt.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}
