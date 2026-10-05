package api_tests

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/util"
)

func TestDigestAuthSuccess(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "digest-auth/auth/dave/diamond",
		Headers: map[string][]string{
			"Cookie":        {"nonce=d9fc96d7fe39099441042eea21006d77"},
			"Authorization": {"Digest username=\"dave\", realm=\"httpbun realm\", nonce=\"d9fc96d7fe39099441042eea21006d77\", uri=\"/digest-auth/auth/dave/diamond\", algorithm=MD5, response=\"57bca3e3d15d4a5123275ce36347ed00\", opaque=\"362d9b0fe6787b534eb27677f4210b61\", qop=auth, nc=00000001, cnonce=\"bb2ec71d21a27e19\""},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.NotContains(resp.Header, "Set-Cookie")
	s.JSONEq(`{
		"authenticated": true,
		"user": "dave"
	}`, body)
}

func TestDigestAuthWithoutCredsRequireCookie(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "digest-auth/auth/dave/diamond?require-cookie=true",
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	match := regexp.MustCompile("\\bnonce=(\\S+)").FindStringSubmatch(resp.Header.Get("Set-Cookie"))
	if !s.NotEmpty(match, "cookie match: "+resp.Header.Get("Set-Cookie")) {
		return
	}
	nonce := match[1]
	m := regexp.MustCompile(
		"Digest realm=\"httpbun realm\", qop=\"auth\", nonce=\"" + nonce + "\", opaque=\"[a-z0-9]+\", algorithm=MD5, stale=FALSE",
	).FindString(resp.Header.Get(c.WWWAuthenticate))
	s.NotEmpty(m, "Unexpected value for "+c.WWWAuthenticate+": "+resp.Header.Get(c.WWWAuthenticate))
	s.JSONEq(`{
		"authenticated": false,
		"token": "",
		"error": "missing authorization header"
	}`, body)
}

func TestDigestAuthWithoutCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "digest-auth/auth/dave/diamond",
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	s.Empty(resp.Header.Get("Set-Cookie"))
	m := regexp.MustCompile(
		"Digest realm=\"httpbun realm\", qop=\"auth\", nonce=\"[a-z0-9]+\", opaque=\"[a-z0-9]+\", algorithm=MD5, stale=FALSE",
	).FindString(resp.Header.Get(c.WWWAuthenticate))
	s.NotEmpty(m, "Unexpected value for "+c.WWWAuthenticate+": "+resp.Header.Get(c.WWWAuthenticate))
	s.JSONEq(`{
		"authenticated": false,
		"token": "",
		"error": "missing authorization header"
	}`, body)
}

func TestDigestAuthWithIncorrectCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "digest-auth/auth/dave/diamond?require-cookie=true",
		Headers: map[string][]string{
			"Cookie":        {"nonce=0801ff8cf72e952e08643d2dc735231d"},
			"Authorization": {"Authorization: Digest username=\"dave2\", realm=\"httpbun realm\", nonce=\"0801ff8cf72e952e08643d2dc735231d\", uri=\"/digest-auth/auth/dave/diamond\", algorithm=MD5, response=\"72cdee27bacbfa650470d0428fe7c4e8\", opaque=\"74061f9b6361455b1a7a74c5b075fd98\", qop=auth, nc=00000001, cnonce=\"810eae48ae823e66\""},
		},
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	match := regexp.MustCompile("\\bnonce=(\\S+)").FindStringSubmatch(resp.Header.Get("Set-Cookie"))
	if !s.NotEmpty(match) {
		return
	}
	nonce := match[1]
	m := regexp.MustCompile(
		"Digest realm=\"httpbun realm\", qop=\"auth\", nonce=\"" + nonce + "\", opaque=\"[a-z0-9]+\", algorithm=MD5, stale=FALSE",
	).FindString(resp.Header.Get(c.WWWAuthenticate))
	s.NotEmpty(m, "Unexpected value for "+c.WWWAuthenticate+": "+resp.Header.Get(c.WWWAuthenticate))
	s.Contains(body, "Response code mismatch")
}

func TestDigestAuthWithIncorrectCredsWithoutCookie(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "digest-auth/auth/dave/diamond",
		Headers: map[string][]string{
			"Authorization": {"Authorization: Digest username=\"dave2\", realm=\"httpbun realm\", nonce=\"0801ff8cf72e952e08643d2dc735231d\", uri=\"/digest-auth/auth/dave/diamond\", algorithm=MD5, response=\"72cdee27bacbfa650470d0428fe7c4e8\", opaque=\"74061f9b6361455b1a7a74c5b075fd98\", qop=auth, nc=00000001, cnonce=\"810eae48ae823e66\""},
		},
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	s.Empty(resp.Header.Get("Set-Cookie"))
	m := regexp.MustCompile(
		"Digest realm=\"httpbun realm\", qop=\"auth\", nonce=\"[a-z0-9]+\", opaque=\"[a-z0-9]+\", algorithm=MD5, stale=FALSE",
	).FindString(resp.Header.Get(c.WWWAuthenticate))
	s.NotEmpty(m, "Unexpected value for "+c.WWWAuthenticate+": "+resp.Header.Get(c.WWWAuthenticate))
	s.Contains(body, "Response code mismatch")
}

// digestHandshake does a full digest auth exchange like a real client: it requests the path, reads the challenge, then
// retries with credentials, signing the request target exactly as sent.
func digestHandshake(t *testing.T, path, username, password string) (http.Response, string) {
	challenge, _ := ExecRequest(R{Path: path})
	if !assert.Equal(t, http.StatusUnauthorized, challenge.StatusCode) {
		return challenge, ""
	}
	nonce := regexp.MustCompile(`nonce="([^"]+)"`).FindStringSubmatch(challenge.Header.Get(c.WWWAuthenticate))[1]

	uri := "/" + path
	ha1 := util.Md5sum(username + ":httpbun realm:" + password)
	ha2 := util.Md5sum("GET:" + uri)
	response := util.Md5sum(ha1 + ":" + nonce + ":00000001:abc:auth:" + ha2)

	headers := map[string][]string{
		"Authorization": {`Digest username="` + username + `", realm="httpbun realm", nonce="` + nonce + `", uri="` + uri +
			`", algorithm=MD5, response="` + response + `", qop=auth, nc=00000001, cnonce="abc"`},
	}
	if cookie := challenge.Header.Get("Set-Cookie"); cookie != "" {
		headers["Cookie"] = []string{strings.Split(cookie, ";")[0]}
	}
	return ExecRequest(R{Path: path, Headers: headers})
}

func TestDigestAuthHandshake(t *testing.T) {
	for _, tt := range []struct{ path, username string }{
		{"digest-auth/auth/dave/diamond", "dave"},
		{"digest-auth/auth/dave/diamond?x=1", "dave"},
		{"digest-auth/auth/dave/diamond?require-cookie=1", "dave"},
		{"digest-auth/auth/da%20ve/diamond", "da ve"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			resp, body := digestHandshake(t, tt.path, tt.username, "diamond")
			assert.Equal(t, http.StatusOK, resp.StatusCode, body)
		})
	}
}
