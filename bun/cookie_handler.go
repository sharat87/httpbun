package bun

import (
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/util"
	"net/http"
	"time"
)

func handleCookies(ex *exchange.Exchange) {
	items := make(map[string]string)
	for _, cookie := range ex.Request.Cookies() {
		items[cookie.Name] = cookie.Value
	}
	util.WriteJson(ex.ResponseWriter, map[string]interface{}{
		"cookies": items,
	})
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

	ex.Redirect(ex.ResponseWriter, "/cookies", true)
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

	ex.Redirect(ex.ResponseWriter, "/cookies", true)
}
