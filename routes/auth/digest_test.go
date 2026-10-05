package auth

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/server/spec"
)

func TestComputeDigestAuthResponse(t *testing.T) {
	s := assert.New(t)
	fakeEx := ex.New(
		nil,
		&http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Path: "/digest-auth/auth/user/pass"},
		},
		spec.Spec{},
	)

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

func TestComputeDigestAuthIntResponse(t *testing.T) {
	s := assert.New(t)
	fakeEx := ex.New(
		nil,
		&http.Request{
			Method: http.MethodPost,
			URL:    &url.URL{Path: "/digest-auth/auth-int/user/pass"},
			Body:   io.NopCloser(strings.NewReader("test body")),
		},
		spec.Spec{},
	)

	response, err := computeDigestAuthResponse(
		"user",
		"pass",
		"dcd98b7102dd2f0e8b11d0f600bfb0c093",
		"00000001",
		"0a4f113b",
		"auth-int",
		fakeEx,
	)

	s.NoError(err)
	s.Equal("35180445afd3b78f781dc5fc50f2204f", response)
	// Hashing the body must not consume it.
	s.Equal("test body", fakeEx.BodyString())
}
