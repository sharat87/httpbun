package api_tests

import (
	"github.com/sharat87/httpbun/c"
	"github.com/stretchr/testify/assert"
	"net/http"
	"regexp"
	"testing"
)

func TestDigestAuthSuccess(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "digest-auth/auth/dave/diamond",
		Headers: map[string][]string{
			"Cookie":        {"nonce=d9fc96d7fe39099441042eea21006d77"},
			"Authorization": {"Digest username=\"dave\", realm=\"testrealm@host.com\", nonce=\"d9fc96d7fe39099441042eea21006d77\", uri=\"/digest-auth/auth/dave/diamond\", algorithm=MD5, response=\"10c1132a06ac0de7c39a07e8553f0f14\", opaque=\"362d9b0fe6787b534eb27677f4210b61\", qop=auth, nc=00000001, cnonce=\"bb2ec71d21a27e19\""},
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

func TestDigestAuthWithoutCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "digest-auth/auth/dave/diamond",
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	match := regexp.MustCompile("\\bnonce=(\\S+)").FindStringSubmatch(resp.Header.Get("Set-Cookie"))
	if !s.NotEmpty(match) {
		return
	}
	nonce := match[1]
	m := regexp.MustCompile(
		"Digest realm=\"testrealm@host.com\", qop=\"auth,auth-int\", nonce=\"" + nonce + "\", opaque=\"[a-z0-9]+\", algorithm=MD5, stale=FALSE",
	).FindString(resp.Header.Get(c.WWWAuthenticate))
	s.NotEmpty(m, "Unexpected value for "+c.WWWAuthenticate)
	s.Equal("", body)
}

func TestDigestAuthWithIncorrectCreds(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(R{
		Path: "digest-auth/auth/dave/diamond",
		Headers: map[string][]string{
			"Cookie":        {"nonce=0801ff8cf72e952e08643d2dc735231d"},
			"Authorization": {"Authorization: Digest username=\"dave2\", realm=\"testrealm@host.com\", nonce=\"0801ff8cf72e952e08643d2dc735231d\", uri=\"/digest-auth/auth/dave/diamond\", algorithm=MD5, response=\"72cdee27bacbfa650470d0428fe7c4e8\", opaque=\"74061f9b6361455b1a7a74c5b075fd98\", qop=auth, nc=00000001, cnonce=\"810eae48ae823e66\""},
		},
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	match := regexp.MustCompile("\\bnonce=(\\S+)").FindStringSubmatch(resp.Header.Get("Set-Cookie"))
	if !s.NotEmpty(match) {
		return
	}
	nonce := match[1]
	m := regexp.MustCompile(
		"Digest realm=\"testrealm@host.com\", qop=\"auth,auth-int\", nonce=\"" + nonce + "\", opaque=\"[a-z0-9]+\", algorithm=MD5, stale=FALSE",
	).FindString(resp.Header.Get(c.WWWAuthenticate))
	s.NotEmpty(m, "Unexpected value for "+c.WWWAuthenticate)
	s.Contains(body, "Error: \"Response code mismatch\"")
}
