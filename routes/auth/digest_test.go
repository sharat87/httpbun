package auth

import (
	"github.com/sharat87/httpbun/exchange"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestComputeDigestAuthResponse(t *testing.T) {
	fakeEx := &exchange.Exchange{
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/digest-auth/auth/user/pass"},
		},
	}

	assert.Equal(t, fakeEx.BodyString(), "")

	response, err := computeDigestAuthResponse(
		"user",
		"pass",
		"dcd98b7102dd2f0e8b11d0f600bfb0c093",
		"00000001",
		"0a4f113b",
		"auth",
		fakeEx,
	)

	assert.NoError(t, err)
	assert.Equal(
		t,
		"dce226046ab3ff3eed7e033afddd0d32",
		response,
	)
}

func TestComputeDigestAuthIntResponse(t *testing.T) {
	fakeEx := &exchange.Exchange{
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/digest-auth/auth-int/user/pass"},
		},
	}

	assert.Equal(t, fakeEx.BodyString(), "")

	response, err := computeDigestAuthResponse(
		"user",
		"pass",
		"dcd98b7102dd2f0e8b11d0f600bfb0c093",
		"00000001",
		"0a4f113b",
		"auth",
		fakeEx,
	)

	assert.NoError(t, err)
	assert.Equal(
		t,
		"eb5e13db29633478dacd26d232602146",
		response,
	)
}
