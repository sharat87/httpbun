package cookies

import (
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/util"
	"net/http"
	"time"
)

var Routes = map[string]exchange.HandlerFn{
	"/cookies":        handleCookies,
	"/cookies/delete": handleCookiesDelete,
	"/cookies/set(/(?P<name>[^/]+)/(?P<value>[^/]+))?": handleCookiesSet,
}

func handleCookies(ex *exchange.Exchange) {
	items := make(map[string]string)
	for _, cookie := range ex.Request.Cookies() {
		items[cookie.Name] = cookie.Value
	}
	util.WriteJson(ex.ResponseWriter, items)
}

func handleCookiesDelete(ex *exchange.Exchange) {
	for name := range ex.Request.URL.Query() {
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:    name,
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
			MaxAge:  -1, // This will produce `Max-Age: 0` in the cookie.
		})
	}

	ex.Redirect("/cookies")
}

func handleCookiesSet(ex *exchange.Exchange) {
	if ex.Field("name") == "" {
		for name, values := range ex.Request.URL.Query() {
			http.SetCookie(ex.ResponseWriter, &http.Cookie{
				Name:  name,
				Value: values[0],
				Path:  "/",
			})
		}

	} else {
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  ex.Field("name"),
			Value: ex.Field("value"),
			Path:  "/",
		})

	}

	ex.Redirect("/cookies")
}
