package auth

import (
	"net/http"
	"testing"

	"github.com/sharat87/httpbun/c"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/util"
)

func TestBearerEmpty(t *testing.T) {
	s := assert.New(t)

	resp := ex.InvokeHandlerForTest(
		"bearer",
		http.Request{},
		BearerAuthRoute,
		handleAuthBearer,
	)

	s.Equal(404, resp.Status)
	s.Equal(0, len(resp.Header))
	s.Greater(len(resp.Body.(string)), 0)
}

func TestBearerFieldParsing(t *testing.T) {
	fields, isMatch := util.MatchRoutePat(ex.MakePat(BearerAuthRoute), "/bearer/dummy_token")

	s := assert.New(t)
	s.True(isMatch)
	s.Equal("dummy_token", fields["tok"])
	s.Equal(1, len(fields))
}

func TestBearerFieldParsingWithTrailingSlash(t *testing.T) {
	fields, isMatch := util.MatchRoutePat(ex.MakePat(BearerAuthRoute), "/bearer/dummy_token/")

	s := assert.New(t)
	s.True(isMatch)
	s.Equal("dummy_token", fields["tok"])
	s.Equal(1, len(fields))
}

func TestBearerFieldParsingWithSpecialChars(t *testing.T) {
	fields, isMatch := util.MatchRoutePat(ex.MakePat(BearerAuthRoute), "/bearer/spe%20cial@token#123%24%25")

	s := assert.New(t)
	s.True(isMatch)
	s.Equal("spe cial@token#123$%", fields["tok"])
	s.Equal(1, len(fields))
}

func TestValidBearerAuth(t *testing.T) {
	s := assert.New(t)

	resp := ex.InvokeHandlerForTest(
		"bearer/dummy_token",
		http.Request{
			Header: http.Header{
				"Authorization": {"Bearer dummy_token"},
			},
		},
		BearerAuthRoute,
		handleAuthBearer,
	)

	s.Equal(0, resp.Status)
	s.Equal(true, resp.Body.(map[string]any)["authenticated"])
}

func TestValidBearerAuthWithSpecialChars(t *testing.T) {
	s := assert.New(t)

	resp := ex.InvokeHandlerForTest(
		"bearer/spe%20cial@token%23123%24%25",
		http.Request{
			Header: http.Header{
				"Authorization": {"Bearer spe cial@token#123$%"},
			},
		},
		BearerAuthRoute,
		handleAuthBearer,
	)

	s.Equal(0, resp.Status)
	s.Equal(true, resp.Body.(map[string]any)["authenticated"])
}

func TestMissingBearerAuthHeader(t *testing.T) {
	s := assert.New(t)

	resp := ex.InvokeHandlerForTest(
		"bearer/dummy_token",
		http.Request{},
		BearerAuthRoute,
		handleAuthBearer,
	)

	s.Equal(401, resp.Status)
	s.Equal("Bearer realm=\"httpbun realm\"", resp.Header.Get(c.WWWAuthenticate))
}

func TestBearerAuthSchemeIsCaseInsensitive(t *testing.T) {
	s := assert.New(t)

	resp := ex.InvokeHandlerForTest(
		"bearer/dummy_token",
		http.Request{
			Header: http.Header{
				"Authorization": {"bearer dummy_token"},
			},
		},
		BearerAuthRoute,
		handleAuthBearer,
	)

	s.Equal(0, resp.Status)
}

func TestWrongBearerToken(t *testing.T) {
	for _, header := range []string{"Bearer nope", "Bearer ", "Bearer", "Basic dummy_token"} {
		t.Run(header, func(t *testing.T) {
			resp := ex.InvokeHandlerForTest(
				"bearer/dummy_token",
				http.Request{
					Header: http.Header{
						"Authorization": {header},
					},
				},
				BearerAuthRoute,
				handleAuthBearer,
			)

			assert.Equal(t, 401, resp.Status)
			assert.Equal(t, "Bearer realm=\"httpbun realm\"", resp.Header.Get(c.WWWAuthenticate))
		})
	}
}
