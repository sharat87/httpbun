package auth

import (
	"net/http"
	"net/url"
	"testing"

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
	fakeEx := &ex.Exchange{
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/digest-auth/auth/user/pass"},
		},
	}

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
	fakeEx := &ex.Exchange{
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/digest-auth/auth-int/user/pass"},
		},
	}

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
