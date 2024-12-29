package auth

import (
	"encoding/base64"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/util"
	"github.com/stretchr/testify/assert"
	"net/http"
	"regexp"
	"testing"
)

func TestFieldParsing(t *testing.T) {
	fields, isMatch := util.MatchRoutePat(*regexp.MustCompile(BasicAuthRoute), "/basic-auth/jam/bread/")

	s := assert.New(t)
	s.True(isMatch)
	s.Equal("jam", fields["user"])
	s.Equal("bread", fields["pass"])
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
