package mix

import (
	"github.com/sharat87/httpbun/exchange"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestMixEmpty(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix",
		http.Request{
			Method: http.MethodGet,
		},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Equal(nil, resp.Body)
}

func TestMixStatusAndBody(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/s=200/b64=b2s=",
		http.Request{
			Method: http.MethodGet,
		},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(200, resp.Status)
	s.Equal([]byte("ok"), resp.Body)
}

func TestMixOnlyBody(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/b64=b2s=",
		http.Request{
			Method: http.MethodGet,
		},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Equal([]byte("ok"), resp.Body)
}
