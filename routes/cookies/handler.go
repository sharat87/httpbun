package cookies

import (
	"net/http"
	"time"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
)

const (
	CookiesRoute       = `/cookies?`
	CookiesDeleteRoute = `/cookies?/delete`
	CookiesSetRoute    = `/cookies?/set(/(?P<name>[^/]+)/(?P<value>[^/]+))?`
)

var RouteList = []ex.Route{
	ex.NewRoute(CookiesRoute, handleCookies),
	ex.NewRoute(CookiesDeleteRoute, handleCookiesDelete),
	ex.NewRoute(CookiesSetRoute, handleCookiesSet),
}

func handleCookies(ex *ex.Exchange) response.Response {
	items := make(map[string]string)
	for _, cookie := range ex.Request.Cookies() {
		items[cookie.Name] = cookie.Value
	}
	return response.Response{
		Body: map[string]any{"cookies": items},
	}
}

func handleCookiesDelete(ex *ex.Exchange) response.Response {
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

	if err := validateCookies(res.Cookies); err != nil {
		return response.BadRequest("%s", err.Error())
	}

	return *res
}

func handleCookiesSet(ex *ex.Exchange) response.Response {
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

	if err := validateCookies(cookies); err != nil {
		return response.BadRequest("%s", err.Error())
	}

	res := ex.RedirectResponse("/cookies")
	res.Cookies = cookies

	return *res
}

// validateCookies checks cookies are valid to send, since Go writes an invalid cookie as an empty Set-Cookie header,
// and silently drops invalid characters in values.
func validateCookies(cookies []http.Cookie) error {
	for _, cookie := range cookies {
		if err := cookie.Valid(); err != nil {
			return err
		}
	}
	return nil
}
