package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/server/spec"
)

func TestBearerAuthMissingExpectedToken(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Path: "bearer",
	})
	s.Equal(http.StatusNotFound, resp.StatusCode)
	s.NotContains(resp.Header, c.WWWAuthenticate)
	s.Equal(
		"Missing expected token. Put the token you expect in the URL, like /bearer/my-token, and send it in the"+
			" Authorization header. For example:\n\n"+
			"    curl -H 'Authorization: Bearer my-token' http://httpbun.test/bearer/my-token\n\n"+
			"A matching token gets a 200, and a missing or wrong one gets a 401.\n",
		body,
	)
}

func TestBearerAuthMissingExpectedTokenWithPathPrefix(t *testing.T) {
	s := assert.New(t)
	server := NewServer(t, spec.Spec{PathPrefix: "/prefix"})
	resp, body := server.Exec(t, R{
		Path: "prefix/bearer",
	})
	s.Equal(http.StatusNotFound, resp.StatusCode)
	s.NotContains(resp.Header, c.WWWAuthenticate)
	s.Contains(body, "    curl -H 'Authorization: Bearer my-token' http://httpbun.test/prefix/bearer/my-token\n")
}

func TestBearerAuthSuccess(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Path: "bearer/dummy_token",
		Headers: map[string][]string{
			"Authorization": {"Bearer dummy_token"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.NotContains(resp.Header, c.WWWAuthenticate)
	s.JSONEq(`{
		"authenticated": true,
		"token": "dummy_token"
	}`, body)
}

func TestBearerAuthSuccessWithSpecialChars(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Path: "bearer/spe%20cial@token%23123%24%25",
		Headers: map[string][]string{
			"Authorization": {"Bearer spe cial@token#123$%"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.NotContains(resp.Header, c.WWWAuthenticate)
	s.JSONEq(`{
		"authenticated": true,
		"token": "spe cial@token#123$%"
	}`, body)
}

func TestBearerAuthSchemeIsCaseInsensitive(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Path: "bearer/dummy_token",
		Headers: map[string][]string{
			"Authorization": {"bearer dummy_token"},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(c.ApplicationJSON, resp.Header.Get(c.ContentType))
	s.JSONEq(`{
		"authenticated": true,
		"token": "dummy_token"
	}`, body)
}

func TestBearerAuthMissingHeader(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{
		Path: "bearer/dummy_token",
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	s.NotContains(resp.Header, c.ContentType)
	s.Equal("Bearer realm=\"httpbun realm\"", resp.Header.Get(c.WWWAuthenticate))
	s.Equal("", body)
}

func TestBearerAuthWrongToken(t *testing.T) {
	for _, header := range []string{"Bearer nope", "Bearer ", "Bearer", "Basic dummy_token", "Bearer dummy_token2"} {
		t.Run(header, func(t *testing.T) {
			s := assert.New(t)
			resp, body := ExecRequest(t, R{
				Path: "bearer/dummy_token",
				Headers: map[string][]string{
					"Authorization": {header},
				},
			})
			s.Equal(http.StatusUnauthorized, resp.StatusCode)
			s.NotContains(resp.Header, c.ContentType)
			s.Equal("Bearer realm=\"httpbun realm\"", resp.Header.Get(c.WWWAuthenticate))
			s.Equal("", body)
		})
	}
}
