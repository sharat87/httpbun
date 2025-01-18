package auth

import (
	"github.com/sharat87/httpbun/exchange"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/url"
	"testing"
)

type DigestSuite struct {
	suite.Suite
}

func TestDigestSuite(t *testing.T) {
	suite.Run(t, new(DigestSuite))
}

func (s *DigestSuite) TestDigestAuthEmpty() {
	resp := exchange.InvokeHandlerForTest(
		"digest-auth",
		http.Request{},
		DigestAuthRoute,
		Routes[DigestAuthRoute],
	)

	s.Equal(404, resp.Status)
	s.Equal(0, len(resp.Header))
	s.Equal("missing/non-empty username/password, use /digest-auth/<username>/<password> instead", resp.Body.(string))
}

func (s *DigestSuite) TestDigestAuthWithValidUsernameAndPasswordButMissingCredentials() {
	resp := exchange.InvokeHandlerForTest(
		"digest-auth/hammer/nails",
		http.Request{},
		DigestAuthRoute,
		Routes[DigestAuthRoute],
	)

	s.Equal(401, resp.Status)
	s.Equal(1, len(resp.Header))
	s.Regexp("^Digest realm=\"httpbun realm\", qop=\"auth\",", resp.Header.Get("WWW-Authenticate"))
	s.Equal(map[string]any{"authenticated": false, "token": "", "error": "missing authorization header"}, resp.Body)
}

func (s *DigestSuite) TestComputeDigestAuthResponse() {
	fakeEx := &exchange.Exchange{
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
	s.Equal("dce226046ab3ff3eed7e033afddd0d32", response)
}

func (s *DigestSuite) TestComputeDigestAuthIntResponse() {
	fakeEx := &exchange.Exchange{
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
	s.Equal("eb5e13db29633478dacd26d232602146", response)
}
