package bun

import (
	"encoding/base64"
	tu "github.com/sharat87/httpbun/test_utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestAuthBasicSuccess(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "basic-auth/scott/tiger",
		Headers: map[string][]string{
			"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("scott:tiger"))},
		},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal("", resp.Header.Get("WWW-Authenticate"))
	s.JSONEq(`{
		"authenticated": true,
		"user": "scott"
	}`, body)
}

func TestAuthBasicFail(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "basic-auth/scott/tiger",
		Headers: map[string][]string{
			"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("tom:lion"))},
		},
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal("Basic realm=\"Fake Realm\"", resp.Header.Get("WWW-Authenticate"))
	s.JSONEq(`{
		"authenticated": false,
		"user": "tom"
	}`, body)
}

func TestAuthBasicMissing(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecTestRequest(tu.R{
		Method: "GET",
		Path:   "basic-auth/scott/tiger",
	})
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))
	s.Equal("Basic realm=\"Fake Realm\"", resp.Header.Get("WWW-Authenticate"))
	s.JSONEq(`{
		"authenticated": false,
		"user": ""
	}`, body)
}
