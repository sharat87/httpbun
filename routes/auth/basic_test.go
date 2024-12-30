package auth

import (
	"encoding/base64"
	"github.com/sharat87/httpbun/c"
	"net/http"
	"testing"

	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/util"
	"github.com/stretchr/testify/assert"
)

func TestFieldParsing(t *testing.T) {
	fields, isMatch := util.MatchRoutePat(BasicAuthRoute, "/basic-auth/jam/bread")

	s := assert.New(t)
	s.True(isMatch)
	s.Equal("jam", fields["user"])
	s.Equal("bread", fields["pass"])
	s.Equal(2, len(fields))
}

func TestFieldParsingWithTrailingSlash(t *testing.T) {
	fields, isMatch := util.MatchRoutePat(BasicAuthRoute, "/basic-auth/jam/bread/")

	s := assert.New(t)
	s.True(isMatch)
	s.Equal("jam", fields["user"])
	s.Equal("bread", fields["pass"])
	s.Equal(2, len(fields))
}

func TestFieldParsingWithSpecialChars(t *testing.T) {
	fields, isMatch := util.MatchRoutePat(BasicAuthRoute, "/basic-auth/user@example.com/p@ssw0rd")

	s := assert.New(t)
	s.True(isMatch)
	s.Equal("user@example.com", fields["user"])
	s.Equal("p@ssw0rd", fields["pass"])
	s.Equal(2, len(fields))
}

func TestFieldParsingWithUrlEncodedChars(t *testing.T) {
	fields, isMatch := util.MatchRoutePat(BasicAuthRoute, "/basic-auth/hello%20world/pass%2Fword%21")

	s := assert.New(t)
	s.True(isMatch)
	s.Equal("hello world", fields["user"])
	s.Equal("pass/word!", fields["pass"])
	s.Equal(2, len(fields))
}

func TestFieldParsingNoMatch(t *testing.T) {
	_, isMatch := util.MatchRoutePat(BasicAuthRoute, "/basic-auth/")

	s := assert.New(t)
	s.False(isMatch)
}

func TestFieldParsingInvalidPath(t *testing.T) {
	_, isMatch := util.MatchRoutePat(BasicAuthRoute, "/wrong-path/user/pass")

	s := assert.New(t)
	s.False(isMatch)
}

func TestValidBasicAuth(t *testing.T) {
	s := assert.New(t)

	ex := exchange.NewForTest(
		http.Request{
			Header: http.Header{
				"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("jam:bread"))},
			},
		},
		map[string]string{"user": "jam", "pass": "bread"},
	)

	resp := handleAuthBasic(&ex)

	s.Equal(0, resp.Status)
}

func TestValidBasicAuthWithSpecialChars(t *testing.T) {
	s := assert.New(t)

	ex := exchange.NewForTest(
		http.Request{
			Header: http.Header{
				"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("hello world@example.com:p@ss/w0rd!"))},
			},
		},
		map[string]string{"user": "hello world@example.com", "pass": "p@ss/w0rd!"},
	)

	resp := handleAuthBasic(&ex)

	s.Equal(0, resp.Status)
}

func TestMissingAuthHeader(t *testing.T) {
	s := assert.New(t)

	ex := exchange.NewForTest(
		http.Request{},
		map[string]string{},
	)

	resp := handleAuthBasic(&ex)

	s.Equal(401, resp.Status)
	s.Equal("Basic realm=\"Fake Realm\"", resp.Header.Get(c.WWWAuthenticate))
}
