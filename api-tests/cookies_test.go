package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/server/spec"
)

const deletedFooCookie = "foo=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0"

func TestGetCookies(t *testing.T) {
	for _, tt := range []struct {
		path    string
		cookies []string
		want    string
	}{
		{"cookie", []string{"foo=bar"}, `{"cookies": {"foo": "bar"}}`},
		{"cookies", []string{"foo=bar", "baz=qux"}, `{"cookies": {"foo": "bar", "baz": "qux"}}`},
		{"cookies", []string{"foo=bar; baz=qux"}, `{"cookies": {"foo": "bar", "baz": "qux"}}`},
		{"cookies", nil, `{"cookies": {}}`},
	} {
		t.Run(tt.path, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path:    tt.path,
				Headers: map[string][]string{"Cookie": tt.cookies},
			})
			s.Equal(http.StatusOK, resp.StatusCode)
			s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
			s.Empty(resp.Header.Values("Set-Cookie"))
			s.JSONEq(tt.want, body)
		})
	}
}

func TestDeleteCookies(t *testing.T) {
	// `?foo=1` and a bare `?foo` both name the cookie to delete.
	for _, path := range []string{"cookies/delete?foo=1", "cookies/delete?foo", "cookie/delete?foo"} {
		t.Run(path, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path:    path,
				Headers: map[string][]string{"Cookie": {"foo=bar", "baz=qux"}},
			})
			s.Equal(http.StatusFound, resp.StatusCode)
			s.Equal("/cookies", resp.Header.Get(c.Location))
			s.Equal([]string{deletedFooCookie}, resp.Header.Values("Set-Cookie"))
			if cookies := resp.Cookies(); s.Len(cookies, 1) {
				s.Equal("foo", cookies[0].Name)
				s.Equal("", cookies[0].Value)
				s.Equal(-1, cookies[0].MaxAge)
			}
			s.Contains(body, `<a href="/cookies">`)
		})
	}
}

func TestSetCookies(t *testing.T) {
	for _, tt := range []struct {
		path string
		want []string
	}{
		{"cookies/set/foo/bar", []string{"foo=bar; Path=/"}},
		{"cookie/set/foo/bar", []string{"foo=bar; Path=/"}},
		{"cookies/set?foo=bar", []string{"foo=bar; Path=/"}},
		{"cookies/set?foo=bar&baz=qux", []string{"foo=bar; Path=/", "baz=qux; Path=/"}},
	} {
		t.Run(tt.path, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{Path: tt.path})
			s.Equal(http.StatusFound, resp.StatusCode)
			s.Equal("/cookies", resp.Header.Get(c.Location))
			s.ElementsMatch(tt.want, resp.Header.Values("Set-Cookie"))
			s.Contains(body, `<a href="/cookies">`)
		})
	}
}

func TestSetCookiesWithPathPrefix(t *testing.T) {
	s := assert.New(t)
	srv := NewServer(t, spec.Spec{PathPrefix: "/mount"})
	resp, _ := srv.Exec(t, R{Path: "mount/cookies/set/foo/bar"})
	s.Equal(http.StatusFound, resp.StatusCode)
	s.Equal("/mount/cookies", resp.Header.Get(c.Location))
	s.Equal([]string{"foo=bar; Path=/"}, resp.Header.Values("Set-Cookie"))
}

func TestSetInvalidCookies(t *testing.T) {
	for _, path := range []string{"cookies/set/a%20b/v", "cookies/set/k/a%3Bb", "cookies/set?a%20b=v&ok=1", "cookies/delete?a%20b"} {
		t.Run(path, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{Path: path})
			s.Equal(http.StatusBadRequest, resp.StatusCode)
			s.Empty(resp.Header.Values("Set-Cookie"))
			s.Empty(resp.Header.Values(c.Location))
			s.NotEmpty(body)
		})
	}
}
