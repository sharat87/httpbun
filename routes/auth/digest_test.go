package auth

import (
	"net/http"
	"net/url"
	"testing"

	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/sharat87/httpbun/server/spec"
	"github.com/stretchr/testify/suite"

	"github.com/sharat87/httpbun/ex"
)

type DigestSuite struct {
	suite.Suite
}

func TestDigestSuite(t *testing.T) {
	suite.Run(t, new(DigestSuite))
}

func (s *DigestSuite) TestDigestAuthEmpty() {
	resp := ex.InvokeHandlerForTest(
		"digest-auth",
		http.Request{},
		DigestAuthRoute,
		handleAuthDigest,
	)

	s.Equal(404, resp.Status)
	s.Equal(0, len(resp.Header))
	s.Equal("missing/non-empty username/password, use /digest-auth/<username>/<password> instead", resp.Body.(string))
}

func (s *DigestSuite) TestDigestAuthWithValidUsernameAndPasswordButMissingCredentials() {
	resp := ex.InvokeHandlerForTest(
		"digest-auth/hammer/nails",
		http.Request{},
		DigestAuthRoute,
		handleAuthDigest,
	)

	s.Equal(401, resp.Status)
	s.Equal(1, len(resp.Header))
	s.Regexp("^Digest realm=\"httpbun realm\", qop=\"auth\",", resp.Header.Get("WWW-Authenticate"))
	s.Equal(map[string]any{"authenticated": false, "token": "", "error": "missing authorization header"}, resp.Body)
}

func (s *DigestSuite) TestComputeDigestAuthResponse() {
	fakeEx := ex.New(
		nil,
		&http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/digest-auth/auth/user/pass"},
		},
		spec.Spec{PathPrefix: ""},
	)

	s.Equal("", fakeEx.BodyString())

	response, err := computeDigestAuthResponse(
		"user",
		"pass",
		"dcd98b7102dd2f0e8b11d0f600bfb0c093",
		"00000001",
		"0a4f113b",
		"auth",
		fakeEx,
	)

	s.NoError(err)
	s.Equal("c5d791b53f3e025c29bb9d812e2ccee1", response)
}

func (s *DigestSuite) TestComputeDigestAuthIntResponse() {
	fakeEx := ex.New(
		nil,
		&http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/digest-auth/auth-int/user/pass"},
		},
		spec.Spec{PathPrefix: ""},
	)

	s.Equal("", fakeEx.BodyString())

	response, err := computeDigestAuthResponse(
		"user",
		"pass",
		"dcd98b7102dd2f0e8b11d0f600bfb0c093",
		"00000001",
		"0a4f113b",
		"auth",
		fakeEx,
	)

	s.NoError(err)
	s.Equal("feb28fc95b61742fa4afd0ad8b630026", response)
}

func (s *DigestSuite) TestDigestAuthWithMultipleQopValues() {
	// 1. First, make a request without credentials to get the nonce from the WWW-Authenticate header.
	resp1 := ex.InvokeHandlerForTest(
		"digest-auth/auth,auth-int/user/pass",
		http.Request{},
		DigestAuthRoute,
		handleAuthDigest,
	)

	s.Equal(401, resp1.Status)
	wwwAuthHeader := resp1.Header.Get("WWW-Authenticate")
	s.NotEmpty(wwwAuthHeader)

	// 2. Parse the WWW-Authenticate header to get the nonce.
	nonceRegex := regexp.MustCompile(`nonce="([^"]+)"`)
	matches := nonceRegex.FindStringSubmatch(wwwAuthHeader)
	s.Len(matches, 2)
	nonce := matches[1]

	// 3. Construct the Authorization header.
	username := "user"
	password := "pass"
	cnonce := "0a4f113b"
	nc := "00000001"
	qop := "auth"
	uri := "/digest-auth/auth,auth-int/user/pass"

	fakeEx := ex.New(
		nil,
		&http.Request{
			Method: "GET",
			URL:    &url.URL{Path: uri},
		},
		spec.Spec{PathPrefix: ""},
	)

	response, err := computeDigestAuthResponse(
		username,
		password,
		nonce,
		nc,
		cnonce,
		qop,
		fakeEx,
	)
	s.NoError(err)

	authHeader := fmt.Sprintf(
		`Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=%s, nc=%s, cnonce="%s", response="%s"`,
		username, REALM, nonce, uri, qop, nc, cnonce, response,
	)

	// 4. Make the second request with the Authorization header.
	req := http.Request{
		Header: http.Header{"Authorization": []string{authHeader}},
		Method: "GET",
	}
	resp2 := ex.InvokeHandlerForTest(
		"digest-auth/auth,auth-int/user/pass",
		req,
		DigestAuthRoute,
		handleAuthDigest,
	)

	s.Equal(200, resp2.Status)
	body, ok := resp2.Body.(map[string]any)
	s.True(ok)
	s.Equal(true, body["authenticated"])
	s.Equal(username, body["user"])
}

func (s *DigestSuite) TestDigestAuthWithAuthIntQop() {
	// 1. First, make a request without credentials to get the nonce from the WWW-Authenticate header.
	resp1 := ex.InvokeHandlerForTest(
		"digest-auth/auth-int/user/pass",
		http.Request{},
		DigestAuthRoute,
		handleAuthDigest,
	)

	s.Equal(401, resp1.Status)
	wwwAuthHeader := resp1.Header.Get("WWW-Authenticate")
	s.NotEmpty(wwwAuthHeader)

	// 2. Parse the WWW-Authenticate header to get the nonce.
	nonceRegex := regexp.MustCompile(`nonce="([^"]+)"`)
	matches := nonceRegex.FindStringSubmatch(wwwAuthHeader)
	s.Len(matches, 2)
	nonce := matches[1]

	// 3. Construct the Authorization header.
	username := "user"
	password := "pass"
	cnonce := "0a4f113b"
	nc := "00000001"
	qop := "auth-int"
	uri := "/digest-auth/auth-int/user/pass"
	body := "test body"

	fakeEx := ex.New(
		nil,
		&http.Request{
			Method: "POST",
			URL:    &url.URL{Path: uri},
			Body:   io.NopCloser(strings.NewReader(body)),
		},
		spec.Spec{PathPrefix: ""},
	)

	response, err := computeDigestAuthResponse(
		username,
		password,
		nonce,
		nc,
		cnonce,
		qop,
		fakeEx,
	)
	s.NoError(err)

	authHeader := fmt.Sprintf(
		`Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=%s, nc=%s, cnonce="%s", response="%s"`,
		username, REALM, nonce, uri, qop, nc, cnonce, response,
	)

	// 4. Make the second request with the Authorization header.
	req := http.Request{
		Header: http.Header{"Authorization": []string{authHeader}},
		Method: "POST",
		Body:   io.NopCloser(strings.NewReader(body)),
	}
	resp2 := ex.InvokeHandlerForTest(
		"digest-auth/auth-int/user/pass",
		req,
		DigestAuthRoute,
		handleAuthDigest,
	)

	s.Equal(200, resp2.Status)
	respBody, ok := resp2.Body.(map[string]any)
	s.True(ok)
	s.Equal(true, respBody["authenticated"])
	s.Equal(username, respBody["user"])

	// Verify that the body is still readable
	s.Equal(body, fakeEx.BodyString())
}
