package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/util"
)

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
