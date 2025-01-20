package auth

import (
	"encoding/base64"
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
				"Authorization": {"Bearer " + base64.StdEncoding.EncodeToString([]byte("dummy_token"))},
			},
		},
		BearerAuthRoute,
		handleAuthBearer,
	)

	s.Equal(0, resp.Status)
}

func TestValidBearerAuthWithSpecialChars(t *testing.T) {
	s := assert.New(t)

	resp := ex.InvokeHandlerForTest(
		"bearer/spe%20cial@token#123%24%25",
		http.Request{
			Header: http.Header{
				"Authorization": {"Bearer " + base64.StdEncoding.EncodeToString([]byte("spe cial@token#123$%"))},
			},
		},
		BearerAuthRoute,
		handleAuthBearer,
	)

	s.Equal(0, resp.Status)
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
