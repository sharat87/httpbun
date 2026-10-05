package cache

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/routes/responses"
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
		{"list", []string{`"other", W/"foo"`}, true},
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
				headers := http.Header{}
				if tt.values != nil {
					headers["If-None-Match"] = tt.values
				}
				resp := ex.InvokeHandlerForTest("etag/foo", http.Request{
					Method:     method,
					Header:     headers,
					RemoteAddr: "127.0.0.1:1234",
				}, `/etag/(?P<etag>[^/]+)`, handleEtag)
				status := resp.Status
				if status == 0 {
					status = http.StatusOK
				}
				wantStatus := http.StatusOK
				if tt.match {
					wantStatus = http.StatusPreconditionFailed
					if method == http.MethodGet || method == http.MethodHead {
						wantStatus = http.StatusNotModified
					}
				}
				if status != wantStatus {
					t.Fatalf("status = %d, want %d", status, wantStatus)
				}
				if got := resp.Header.Get("ETag"); got != `"foo"` {
					t.Fatalf("ETag = %q, want %q", got, `"foo"`)
				}
				if tt.match {
					if resp.Body != nil || resp.Writer != nil {
						t.Fatal("conditional response contains a body")
					}
				} else {
					info, ok := resp.Body.(*responses.Info)
					if !ok || info.Method != method || info.Url != "http://localhost/etag/foo" {
						t.Fatalf("body = %#v, want request information", resp.Body)
					}
				}
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
			for _, conditional := range []bool{false, true} {
				headers := http.Header{}
				wantStatus := http.StatusOK
				if conditional {
					headers.Set("If-None-Match", `"other", W/`+tag)
					wantStatus = http.StatusNotModified
				}
				resp := ex.InvokeHandlerForTest("etag/"+tt.path, http.Request{
					Method:     http.MethodGet,
					Header:     headers,
					RemoteAddr: "127.0.0.1:1234",
				}, `/etag/(?P<etag>[^/]+)`, handleEtag)
				status := resp.Status
				if status == 0 {
					status = http.StatusOK
				}
				if status != wantStatus {
					t.Fatalf("conditional = %t: status = %d, want %d", conditional, status, wantStatus)
				}
				if got := resp.Header.Get("ETag"); got != tag {
					t.Fatalf("ETag = %q, want %q", got, tag)
				}
				if conditional && (resp.Body != nil || resp.Writer != nil) {
					t.Fatal("conditional response contains a body")
				}
			}
		})
	}
}

func TestEtagRejectsInvalidOpaqueValues(t *testing.T) {
	for _, opaque := range []string{"foo bar", `foo"bar`, "foo\tbar", "foo\nbar", "foo\x00bar", "foo\x7fbar"} {
		t.Run(url.PathEscape(opaque), func(t *testing.T) {
			resp := ex.InvokeHandlerForTest("etag/"+url.PathEscape(opaque), http.Request{
				Method:     http.MethodGet,
				RemoteAddr: "127.0.0.1:1234",
			}, `/etag/(?P<etag>[^/]+)`, handleEtag)
			if resp.Status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", resp.Status)
			}
			if got := resp.Header.Get("ETag"); got != "" {
				t.Fatalf("invalid opaque value emitted ETag %q", got)
			}
		})
	}
}

func TestEtagPathPrefix(t *testing.T) {
	for _, path := range []string{"foo+bar", "foo%2Bbar"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://localhost/mount/etag/"+path, nil)
			req.Header.Set("If-None-Match", `"foo+bar"`)
			exchange := ex.New(nil, req, spec.Spec{PathPrefix: "/mount"})
			if !exchange.MatchAndLoadFields(RouteList[2].Pat) {
				t.Fatal("ETag route did not match")
			}
			resp := handleEtag(exchange)
			if resp.Status != http.StatusNotModified {
				t.Fatalf("status = %d, want 304", resp.Status)
			}
			if got := resp.Header.Get("ETag"); got != `"foo+bar"` {
				t.Fatalf("ETag = %q, want %q", got, `"foo+bar"`)
			}
		})
	}
}
