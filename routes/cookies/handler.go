package cookies

import (
	"net/http"
	"time"

	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
)

const (
	CookiesRoute       = `/cookies?`
	CookiesDeleteRoute = `/cookies?/delete`
	CookiesSetRoute    = `/cookies?/set(/(?P<name>[^/]+)/(?P<value>[^/]+))?`
)

var Routes = map[string]exchange.HandlerFn{
	CookiesRoute:       handleCookies,
	CookiesDeleteRoute: handleCookiesDelete,
	CookiesSetRoute:    handleCookiesSet,
}

func handleCookies(ex *exchange.Exchange) response.Response {
	items := make(map[string]string)
	for _, cookie := range ex.Request.Cookies() {
		items[cookie.Name] = cookie.Value
	}
	return response.Response{
		Body: map[string]any{"cookies": items},
	}
}

func handleCookiesDelete(ex *exchange.Exchange) response.Response {
	res := ex.RedirectResponse("/cookies")

	for name := range ex.Request.URL.Query() {
		res.Cookies = append(res.Cookies, http.Cookie{
			Name:    name,
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
			MaxAge:  -1, // This will produce `Max-Age: 0` in the cookie.
		})
	}

	return *res
}

func handleCookiesSet(ex *exchange.Exchange) response.Response {
	var cookies []http.Cookie

	if ex.Field("name") == "" {
		for name, values := range ex.Request.URL.Query() {
			cookies = append(cookies, http.Cookie{
				Name:  name,
				Value: values[0],
				Path:  "/",
			})
		}

	} else {
		cookies = append(cookies, http.Cookie{
			Name:  ex.Field("name"),
			Value: ex.Field("value"),
			Path:  "/",
		})

	}

	res := ex.RedirectResponse("/cookies")
	res.Cookies = cookies

	return *res
}
