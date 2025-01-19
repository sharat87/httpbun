package cookies

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/sharat87/httpbun/exchange"
)

type CookiesSuite struct {
	suite.Suite
}

func TestCookiesSuite(t *testing.T) {
	suite.Run(t, new(CookiesSuite))
}

func (s *CookiesSuite) TestGetCookiesSingular() {
	resp := exchange.InvokeHandlerForTest(
		"cookie",
		http.Request{
			Header: http.Header{
				"Cookie": []string{"foo=bar"},
			},
		},
		CookiesRoute,
		Routes[CookiesRoute],
	)

	s.Equal(0, resp.Status)
	s.Equal(map[string]any{"cookies": map[string]string{"foo": "bar"}}, resp.Body)
}

func (s *CookiesSuite) TestGetCookiesPlural() {
	resp := exchange.InvokeHandlerForTest(
		"cookies",
		http.Request{
			Header: http.Header{
				"Cookie": []string{"foo=bar", "baz=qux"},
			},
		},
		CookiesRoute,
		Routes[CookiesRoute],
	)

	s.Equal(0, resp.Status)
	s.Equal(map[string]any{"cookies": map[string]string{"foo": "bar", "baz": "qux"}}, resp.Body)
}

func (s *CookiesSuite) TestDeleteCookies() {
	resp := exchange.InvokeHandlerForTest(
		"cookies/delete?foo=1",
		http.Request{
			Header: http.Header{
				"Cookie": []string{"foo=bar", "baz=qux"},
			},
		},
		CookiesDeleteRoute,
		Routes[CookiesDeleteRoute],
	)

	s.Equal(302, resp.Status)

	s.Equal(1, len(resp.Header))
	s.Equal("/cookies", resp.Header.Get("Location"))

	s.Equal(1, len(resp.Cookies))
	s.Equal("foo=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0", resp.Cookies[0].String())
}

func (s *CookiesSuite) TestDeleteCookiesSingularAndNoValue() {
	resp := exchange.InvokeHandlerForTest(
		"cookies/delete?foo",
		http.Request{
			Header: http.Header{
				"Cookie": []string{"foo=bar", "baz=qux"},
			},
		},
		CookiesDeleteRoute,
		Routes[CookiesDeleteRoute],
	)

	s.Equal(302, resp.Status)

	s.Equal(1, len(resp.Header))
	s.Equal("/cookies", resp.Header.Get("Location"))

	s.Equal(1, len(resp.Cookies))
	s.Equal("foo=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0", resp.Cookies[0].String())
}

func (s *CookiesSuite) TestSetCookiesWithNameAndValueInPath() {
	resp := exchange.InvokeHandlerForTest(
		"cookies/set/foo/bar",
		http.Request{},
		CookiesSetRoute,
		Routes[CookiesSetRoute],
	)

	s.Equal(302, resp.Status)

	s.Equal(1, len(resp.Header))
	s.Equal("/cookies", resp.Header.Get("Location"))

	s.Equal(1, len(resp.Cookies))
	s.Equal("foo=bar; Path=/", resp.Cookies[0].String())
}

func (s *CookiesSuite) TestSetCookiesWithNameAndValueInQuery() {
	resp := exchange.InvokeHandlerForTest(
		"cookies/set?foo=bar",
		http.Request{},
		CookiesSetRoute,
		Routes[CookiesSetRoute],
	)

	s.Equal(302, resp.Status)

	s.Equal(1, len(resp.Header))
	s.Equal("/cookies", resp.Header.Get("Location"))

	s.Equal(1, len(resp.Cookies))
	s.Equal("foo=bar; Path=/", resp.Cookies[0].String())
}

func (s *CookiesSuite) TestSetCookiesWithNameAndValueInQueryMultiple() {
	resp := exchange.InvokeHandlerForTest(
		"cookies/set?foo=bar&baz=qux",
		http.Request{},
		CookiesSetRoute,
		Routes[CookiesSetRoute],
	)

	s.Equal(302, resp.Status)

	s.Equal(1, len(resp.Header))
	s.Equal("/cookies", resp.Header.Get("Location"))

	s.Equal(2, len(resp.Cookies))
	s.Equal("foo=bar; Path=/", resp.Cookies[0].String())
	s.Equal("baz=qux; Path=/", resp.Cookies[1].String())
}
