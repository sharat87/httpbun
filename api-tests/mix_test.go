package api_tests

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mixHeaders returns the response headers, minus the ones that are present on every response, and vary between runs or
// builds. This lets tests assert the exact set of headers a mix response has.
func mixHeaders(resp http.Response) http.Header {
	h := resp.Header.Clone()
	h.Del("Date")
	h.Del("X-Powered-By")
	return h
}

// plainTextHeaders are the headers of a mix response with the given plain text body, and no other headers.
func plainTextHeaders(body string) http.Header {
	return http.Header{
		"Content-Length": {strconv.Itoa(len(body))},
		"Content-Type":   {"text/plain; charset=utf-8"},
	}
}

func TestMixEmpty(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Path: "mix",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(http.Header{"Content-Length": {"0"}}, mixHeaders(resp))
	s.Empty(resp.Cookies())
	s.Equal("", body)
}

func TestMixStatus(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Method: http.MethodGet,
		Path:   "mix/s=200",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("", body)
	resp, body = ExecRequest(t, R{
		Method: http.MethodPost,
		Path:   "mix/s=200",
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("", body)
}

func TestMixInvalidStatus(t *testing.T) {
	for _, tc := range []struct {
		path string
		body string
	}{
		{"mix/s=", "No status codes given"},
		{"mix/s=abc", "No status codes given"},
		{"mix/s=100", "Invalid status code: 100"},
		{"mix/s=1000", "Invalid status code: 1000"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: tc.path,
			})
			s.Equal(http.StatusBadRequest, resp.StatusCode)
			s.Equal(plainTextHeaders(tc.body), mixHeaders(resp))
			s.Equal(tc.body, body)
		})
	}
}

func TestMixBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		body string
	}{
		{"body only", "mix/b64=c2FtcGxl", "sample"},
		{"short body only", "mix/b64=b2s=", "ok"},
		{"status and body", "mix/s=200/b64=b2s=", "ok"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: tc.path,
			})
			s.Equal(http.StatusOK, resp.StatusCode)
			s.Equal(plainTextHeaders(tc.body), mixHeaders(resp))
			s.Equal(tc.body, body)
		})
	}
}

func TestMixInvalidBase64(t *testing.T) {
	for _, tc := range []struct {
		path string
		body string
	}{
		{"mix/b64=invalid", "illegal base64 data at input byte 4"},
		{"mix/b64=invalid!base64", "illegal base64 data at input byte 7"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: tc.path,
			})
			s.Equal(http.StatusBadRequest, resp.StatusCode)
			s.Equal(plainTextHeaders(tc.body), mixHeaders(resp))
			s.Equal(tc.body, body)
		})
	}
}

func TestMixXSSAttack(t *testing.T) {
	s := assert.New(t)
	xssPayload := "<script>alert('XSS')</script>"
	encodedPayload := "PHNjcmlwdD5hbGVydCgnWFNTJyk8L3NjcmlwdD4="
	resp, body := ExecRequest(t, R{
		Path: "mix/h=Content-Type:text%2Fhtml/b64=" + encodedPayload,
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("text/html", resp.Header.Get("Content-Type"))
	s.Equal(xssPayload, body)
}

func TestMixHeaders(t *testing.T) {
	for _, tc := range []struct {
		name    string
		path    string
		headers http.Header
	}{
		{
			"single header",
			"mix/h=x-one:great",
			http.Header{"X-One": {"great"}},
		},
		{
			"encoded slash",
			"mix/h=x-key:val%2fmore",
			http.Header{"X-Key": {"val/more"}},
		},
		{
			"two headers",
			"mix/h=x-key:val/h=x-key2:val2",
			http.Header{"X-Key": {"val"}, "X-Key2": {"val2"}},
		},
		{
			"special characters",
			"mix/h=x-special-header:value%20with%20spaces/h=content-type:application%2Fjson%3Bcharset%3Dutf-8/h=x-symbols:!%40%23%24%25",
			http.Header{
				"X-Special-Header": {"value with spaces"},
				"Content-Type":     {"application/json;charset=utf-8"},
				"X-Symbols":        {"!@#$%"},
			},
		},
		{
			"repeated header",
			"mix/h=x-multi:value1/h=x-multi:value2/h=x-multi:value3",
			http.Header{"X-Multi": {"value1", "value2", "value3"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: tc.path,
			})
			s.Equal(http.StatusOK, resp.StatusCode)
			tc.headers.Set("Content-Length", "0")
			s.Equal(tc.headers, mixHeaders(resp))
			s.Empty(resp.Cookies())
			s.Equal("", body)
		})
	}
}

func TestMixCookies(t *testing.T) {
	for _, tc := range []struct {
		name       string
		path       string
		setCookies []string
		cookies    map[string]string
	}{
		{
			"single cookie",
			"mix/c=name:content",
			[]string{"name=content; Path=/"},
			map[string]string{"name": "content"},
		},
		{
			"another single cookie",
			"mix/c=session:abc123",
			[]string{"session=abc123; Path=/"},
			map[string]string{"session": "abc123"},
		},
		{
			"two cookies",
			"mix/c=name:content/c=another:more",
			[]string{"name=content; Path=/", "another=more; Path=/"},
			map[string]string{"name": "content", "another": "more"},
		},
		{
			"three cookies",
			"mix/c=cookie1:value1/c=cookie2:value2/c=cookie3:value3",
			[]string{"cookie1=value1; Path=/", "cookie2=value2; Path=/", "cookie3=value3; Path=/"},
			map[string]string{"cookie1": "value1", "cookie2": "value2", "cookie3": "value3"},
		},
		{
			"special characters",
			"mix/c=complex-cookie:value%20with%20spaces%20%26%20symbols%21%40%23%25",
			// Values with spaces are sent quoted.
			[]string{`complex-cookie="value with spaces & symbols!@#%"; Path=/`},
			map[string]string{"complex-cookie": "value with spaces & symbols!@#%"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: tc.path,
			})
			s.Equal(http.StatusOK, resp.StatusCode)
			s.Equal(http.Header{
				"Content-Length": {"0"},
				"Set-Cookie":     tc.setCookies,
			}, mixHeaders(resp))
			cookies := map[string]string{}
			for _, cookie := range resp.Cookies() {
				s.Equal("/", cookie.Path)
				cookies[cookie.Name] = cookie.Value
			}
			s.Equal(tc.cookies, cookies)
			s.Equal("", body)
		})
	}
}

func TestMixDeleteCookie(t *testing.T) {
	for _, tc := range []struct {
		path  string
		names []string
	}{
		{"mix/cd=name", []string{"name"}},
		{"mix/cd=name/cd=another", []string{"name", "another"}},
		{"mix/cd=cookie1/cd=cookie2", []string{"cookie1", "cookie2"}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: tc.path,
			})
			s.Equal(http.StatusOK, resp.StatusCode)
			var setCookies []string
			for _, name := range tc.names {
				setCookies = append(setCookies, name+"=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0")
			}
			s.Equal(http.Header{
				"Content-Length": {"0"},
				"Set-Cookie":     setCookies,
			}, mixHeaders(resp))
			cookies := resp.Cookies()
			if s.Len(cookies, len(tc.names)) {
				for i, name := range tc.names {
					s.Equal(name, cookies[i].Name)
					s.Equal("", cookies[i].Value)
					s.Equal(-1, cookies[i].MaxAge) // `Max-Age=0` is parsed as -1, meaning "delete now".
				}
			}
			s.Equal("", body)
		})
	}
}

func TestMixRedirect(t *testing.T) {
	for _, tc := range []struct {
		name     string
		path     string
		status   int
		location string
	}{
		{
			"http",
			"mix/r=http%3A%2F%2Fexample.com",
			http.StatusTemporaryRedirect,
			"http://example.com",
		},
		{
			"https",
			"mix/r=https%3A%2F%2Fexample.com",
			http.StatusTemporaryRedirect,
			"https://example.com",
		},
		{
			"http with status",
			"mix/s=301/r=http%3A%2F%2Fexample.com",
			http.StatusMovedPermanently,
			"http://example.com",
		},
		{
			"https with status",
			"mix/s=301/r=https%3A%2F%2Fexample.com",
			http.StatusMovedPermanently,
			"https://example.com",
		},
		{
			"query and fragment",
			"mix/r=https%3A%2F%2Fexample.com%2Fpath%3Fkey%3Dvalue%26other%3Dthing%20value%23fragment",
			http.StatusTemporaryRedirect,
			"https://example.com/path?key=value&other=thing value#fragment",
		},
		{
			"keeps encoded characters",
			"mix/r=https%3A%2F%2Fexample.com%2F%3Fq%3Da%252Bb%2Bc",
			http.StatusTemporaryRedirect,
			"https://example.com/?q=a%2Bb+c",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: tc.path,
			})
			s.Equal(tc.status, resp.StatusCode)
			s.Equal(http.Header{
				"Content-Length": {"0"},
				"Location":       {tc.location},
			}, mixHeaders(resp))
			s.Empty(resp.Cookies())
			s.Equal("", body)
		})
	}
}

func TestMixMultipleRedirects(t *testing.T) {
	for _, path := range []string{
		"mix/r=http%3A%2F%2Fexample.com/r=another",
		"mix/r=https%3A%2F%2Fexample1.com/r=https%3A%2F%2Fexample2.com",
	} {
		t.Run(path, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: path,
			})
			s.Equal(http.StatusBadRequest, resp.StatusCode)
			s.Equal(plainTextHeaders("multiple redirects not allowed"), mixHeaders(resp))
			s.Equal("multiple redirects not allowed", body)
		})
	}
}

func TestMixDelay(t *testing.T) {
	s := assert.New(t)
	start := time.Now()
	resp, body := ExecRequest(t, R{
		Path: "mix/d=0.1",
	})
	s.GreaterOrEqual(time.Since(start), 100*time.Millisecond)
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(http.Header{"Content-Length": {"0"}}, mixHeaders(resp))
	s.Equal("", body)
}

func TestMixInvalidDelay(t *testing.T) {
	for _, tc := range []struct {
		path string
		body string
	}{
		{"mix/d=invalid", "invalid delay value: 'invalid'"},
		{"mix/d=-1", "delay must be a positive number"},
		{"mix/d=11", "delay must be less than 10 seconds"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: tc.path,
			})
			s.Equal(http.StatusBadRequest, resp.StatusCode)
			s.Equal(plainTextHeaders(tc.body), mixHeaders(resp))
			s.Equal(tc.body, body)
		})
	}
}

func TestMixTemplate(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Path: "mix/t=" + base64.StdEncoding.EncodeToString([]byte(`length is {{len "abc"}}`)),
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(plainTextHeaders("length is 3"), mixHeaders(resp))
	s.Empty(resp.Cookies())
	s.Equal("length is 3", body)
}

func TestMixInvalidTemplate(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		body string
	}{
		{
			"invalid base64",
			"mix/t=invalid",
			"illegal base64 data at input byte 4",
		},
		{
			"parse error",
			"mix/t=" + base64.StdEncoding.EncodeToString([]byte("{{if}}")),
			"template: mix:1: missing value for if",
		},
		{
			"seq with zero step",
			"mix/t=" + base64.StdEncoding.EncodeToString([]byte("{{range seq 0 5 0}}x{{end}}")),
			`template: mix:1:8: executing "mix" at <seq 0 5 0>: error calling seq: seq step can't be zero`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: tc.path,
			})
			s.Equal(http.StatusBadRequest, resp.StatusCode)
			s.Equal(plainTextHeaders(tc.body), mixHeaders(resp))
			s.Equal(tc.body, body)
		})
	}
}
