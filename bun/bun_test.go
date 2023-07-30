package bun

import (
	tu "github.com/sharat87/httpbun/test_utils"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func ExecTestRequest(request tu.R) (http.Response, string) {
	var bodyReader io.Reader
	if request.Body != "" {
		bodyReader = strings.NewReader(request.Body)
	}

	//goland:noinspection HttpUrlsUsage
	req := httptest.NewRequest(request.Method, "http://example.com/"+request.Path, bodyReader)

	for name, values := range request.Headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	w := httptest.NewRecorder()
	BunHandler.ServeHTTP(w, req)

	resp := w.Result()
	responseBody, _ := io.ReadAll(resp.Body)

	return *resp, string(responseBody)
}

func TestHeaders(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "headers",
		Headers: map[string][]string{
			"X-One": {"custom header value"},
			"X-Two": {"another custom header"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.JSONEq(`{
		"X-One": "custom header value",
		"X-Two": "another custom header"
	}`, body)
}

func TestHeadersRepeat(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "headers",
		Headers: map[string][]string{
			"X-One": {"custom header value", "another custom header"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.JSONEq(`{
		"X-One": "custom header value,another custom header"
	}`, body)
}

func TestBasicAuthWithoutCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "basic-auth/scott/tiger",
	})
	s.Equal(401, resp.StatusCode)
	s.Equal("Basic realm=\"Fake Realm\"", resp.Header.Get("WWW-Authenticate"))
	s.JSONEq(`{
		"authenticated": false,
		"user": ""
	}`, body)
}

func TestBasicAuthWithValidCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "basic-auth/scott/tiger",
		Headers: map[string][]string{
			"Authorization": {"Basic c2NvdHQ6dGlnZXI="},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("", resp.Header.Get("WWW-Authenticate"))
	s.JSONEq(`{
		"authenticated": true,
		"user": "scott"
	}`, body)
}

func TestBasicAuthWithInvalidCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "basic-auth/scott/tiger",
		Headers: map[string][]string{
			"Authorization": {"Basic c2NvdHQ6d3Jvbmc="},
		},
	})
	s.Equal(401, resp.StatusCode)
	s.Equal("Basic realm=\"Fake Realm\"", resp.Header.Get("WWW-Authenticate"))
	s.JSONEq(`{
		"authenticated": false,
		"user": "scott"
	}`, body)
}

func TestBearerAuthWithoutToken(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "bearer",
	})
	s.Equal(401, resp.StatusCode)
	s.Equal("Bearer", resp.Header.Get("WWW-Authenticate"))
	s.Equal(0, len(body))
}

func TestBearerAuthWithToken(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "bearer",
		Headers: map[string][]string{
			"Authorization": {"Bearer my-auth-token"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("", resp.Header.Get("WWW-Authenticate"))
	s.JSONEq(`{
		"authenticated": true,
		"token": "my-auth-token"
	}`, body)
}

func TestDigestAuthWithoutCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
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
	s.Equal(0, len(body))
}

func TestDigestAuthWitCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "digest-auth/auth/dave/diamond",
		Headers: map[string][]string{
			"Cookie":        {"nonce=d9fc96d7fe39099441042eea21006d77"},
			"Authorization": {"Digest username=\"dave\", realm=\"testrealm@host.com\", nonce=\"d9fc96d7fe39099441042eea21006d77\", uri=\"/digest-auth/auth/dave/diamond\", algorithm=MD5, response=\"10c1132a06ac0de7c39a07e8553f0f14\", opaque=\"362d9b0fe6787b534eb27677f4210b61\", qop=auth, nc=00000001, cnonce=\"bb2ec71d21a27e19\""},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Empty(resp.Header.Get("Set-Cookie"))
	s.JSONEq(`{
		"authenticated": true,
		"user": "dave"
	}`, body)
}

func TestDigestAuthWitIncorrectUser(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "digest-auth/auth/dave/diamond",
		Headers: map[string][]string{
			"Cookie":        {"nonce=0801ff8cf72e952e08643d2dc735231d"},
			"Authorization": {"Authorization: Digest username=\"dave2\", realm=\"testrealm@host.com\", nonce=\"0801ff8cf72e952e08643d2dc735231d\", uri=\"/digest-auth/auth/dave/diamond\", algorithm=MD5, response=\"72cdee27bacbfa650470d0428fe7c4e8\", opaque=\"74061f9b6361455b1a7a74c5b075fd98\", qop=auth, nc=00000001, cnonce=\"810eae48ae823e66\""},
		},
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
	// s.Equal(string(body), "")
}

func TestResponseHeaders(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "response-headers?one=two&three=four",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal([]string{"two"}, resp.Header.Values("One"))
	s.Equal([]string{"four"}, resp.Header.Values("Three"))
	s.JSONEq(`{
		"Content-Length": "103",
		"Content-Type": "application/json",
		"One": "two",
		"Three": "four"
	}`, body)
}

func TestResponseHeadersRepeated(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "response-headers?one=two&one=four",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal([]string{"two", "four"}, resp.Header.Values("One"))
	s.JSONEq(`{
		"Content-Length": "106",
		"Content-Type": "application/json",
		"One": [
			"two",
			"four"
		]
	}`, body)
}

func TestDrip(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "drip?duration=1&delay=0",
	})
	s.Equal(200, resp.StatusCode)
	s.Equal(strings.Repeat("*", 10), body)
}

func TestIpInXForwardedFor(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "ip",
		Headers: map[string][]string{
			"X-Httpbun-Forwarded-For": {"12.34.56.78"},
		},
	})
	s.Equal(200, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.JSONEq(`{
		"origin": "12.34.56.78"
	}`, body)
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
