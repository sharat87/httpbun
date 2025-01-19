package mix

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/exchange"
)

func TestMixEmpty(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Nil(resp.Body)
	s.Equal(0, len(resp.Header))
}

func TestMixStatusAndBody(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/s=200/b64=b2s=",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(200, resp.Status)
	s.Equal([]byte("ok"), resp.Body)
	s.Equal(1, len(resp.Header))
}

func TestMixOnlyBody(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/b64=b2s=",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Equal([]byte("ok"), resp.Body)
	s.Equal(1, len(resp.Header))
}

func TestMixInvalidBody(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/b64=invalid",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(400, resp.Status)
	s.Equal("illegal base64 data at input byte 4", resp.Body)
	s.Equal(0, len(resp.Header))
}

func TestMixSingleHeader(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/h=x-one:great",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Equal(nil, resp.Body)
	s.Equal(1, len(resp.Header))
	s.Equal("great", resp.Header.Get("x-one"))
}

func TestMixMultipleHeadersWithSpecialChars(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/h=x-special-header:value%20with%20spaces/h=content-type:application%2Fjson%3Bcharset%3Dutf-8/h=x-symbols:!%40%23%24%25",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Nil(resp.Body)
	s.Equal(3, len(resp.Header))
	s.Equal("value with spaces", resp.Header.Get("x-special-header"))
	s.Equal("application/json;charset=utf-8", resp.Header.Get("content-type"))
	s.Equal("!@#$%", resp.Header.Get("x-symbols"))
}

func TestMixRepeatedHeader(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/h=x-multi:value1/h=x-multi:value2/h=x-multi:value3",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Nil(resp.Body)
	s.Equal(1, len(resp.Header))
	values := resp.Header.Values("x-multi")
	s.Equal(3, len(values))
	s.Equal("value1", values[0])
	s.Equal("value2", values[1])
	s.Equal("value3", values[2])
}

func TestMixSetCookie(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/c=session:abc123",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Nil(resp.Body)
	s.Equal(0, len(resp.Header))
	s.Equal(1, len(resp.Cookies))
	s.Equal("session", resp.Cookies[0].Name)
	s.Equal("abc123", resp.Cookies[0].Value)
	s.Equal("/", resp.Cookies[0].Path)
}

func TestMixSetCookieWithSpecialChars(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/c=complex-cookie:value%20with%20spaces%20%26%20symbols%21%40%23%25",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Nil(resp.Body)
	s.Equal(0, len(resp.Header))
	s.Equal(1, len(resp.Cookies))
	s.Equal("complex-cookie", resp.Cookies[0].Name)
	s.Equal("value with spaces & symbols!@#%", resp.Cookies[0].Value)
	s.Equal("/", resp.Cookies[0].Path)
}

func TestMixMultipleCookies(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/c=cookie1:value1/c=cookie2:value2/c=cookie3:value3",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Nil(resp.Body)
	s.Equal(0, len(resp.Header))
	s.Equal(3, len(resp.Cookies))
	s.Equal("cookie1", resp.Cookies[0].Name)
	s.Equal("value1", resp.Cookies[0].Value)
	s.Equal("cookie2", resp.Cookies[1].Name)
	s.Equal("value2", resp.Cookies[1].Value)
	s.Equal("cookie3", resp.Cookies[2].Name)
	s.Equal("value3", resp.Cookies[2].Value)
}

func TestMixRedirect(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/r=https%3A%2F%2Fexample.com",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(307, resp.Status)
	s.Nil(resp.Body)
	s.Equal(1, len(resp.Header))
	s.Equal("https://example.com", resp.Header.Get("Location"))
	s.Equal(0, len(resp.Cookies))
}

func TestMixRedirectWithCustomStatus(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/s=301/r=https%3A%2F%2Fexample.com",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(301, resp.Status)
	s.Nil(resp.Body)
	s.Equal(1, len(resp.Header))
	s.Equal("https://example.com", resp.Header.Get("Location"))
	s.Equal(0, len(resp.Cookies))
}

func TestMixRedirectWithQueryAndFragment(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/r=https%3A%2F%2Fexample.com%2Fpath%3Fkey%3Dvalue%26other%3Dthing%20value%23fragment",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(307, resp.Status)
	s.Nil(resp.Body)
	s.Equal(1, len(resp.Header))
	s.Equal("https://example.com/path?key=value&other=thing value#fragment", resp.Header.Get("Location"))
	s.Equal(0, len(resp.Cookies))
}

func TestMixMultipleRedirectsError(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/r=https%3A%2F%2Fexample1.com/r=https%3A%2F%2Fexample2.com",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(400, resp.Status)
	s.Equal("multiple redirects not allowed", resp.Body)
}
func TestMixCookieDeletion(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/cd=cookie1/cd=cookie2",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Nil(resp.Body)
	s.Equal(2, len(resp.Cookies))
	s.Equal("cookie1", resp.Cookies[0].Name)
	s.Equal("", resp.Cookies[0].Value)
	s.Equal(-1, resp.Cookies[0].MaxAge)
	s.Equal("cookie2", resp.Cookies[1].Name)
	s.Equal("", resp.Cookies[1].Value)
	s.Equal(-1, resp.Cookies[1].MaxAge)
}

func TestMixDelay(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/d=0.1",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Nil(resp.Body)
	s.Equal(0, len(resp.Header))
}

func TestMixDelayInvalid(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/d=invalid",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(400, resp.Status)
	s.Equal("invalid delay value: 'invalid'", resp.Body)
	s.Equal(0, len(resp.Header))
}

func TestMixDelayNegative(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/d=-1",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(400, resp.Status)
	s.Equal("delay must be a positive number", resp.Body)
	s.Equal(0, len(resp.Header))
}

func TestMixDelayTooLarge(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/d=11",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(400, resp.Status)
	s.Equal("delay must be less than 10 seconds", resp.Body)
	s.Equal(0, len(resp.Header))
}

func TestMixTemplateDirectiveInvalid(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/t=invalid",
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(400, resp.Status)
	s.Equal(0, len(resp.Header))
	s.Equal("illegal base64 data at input byte 4", resp.Body)
	s.Equal(0, len(resp.Cookies))
}

func TestMixTemplateDirective(t *testing.T) {
	s := assert.New(t)

	resp := exchange.InvokeHandlerForTest(
		"mix/t="+base64.StdEncoding.EncodeToString([]byte(`length is {{len "abc"}}`)),
		http.Request{},
		PatMix,
		Routes[PatMix],
	)

	s.Equal(0, resp.Status)
	s.Equal(1, len(resp.Header))
	s.Equal([]byte("length is 3"), resp.Body)
	s.Equal(0, len(resp.Cookies))
}
