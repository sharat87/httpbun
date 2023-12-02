package method

import (
	"github.com/sharat87/httpbun/bun/responses"
	"github.com/sharat87/httpbun/exchange"
	"net/http"
	"strings"
)

var Routes = map[string]exchange.HandlerFn{
	"/get":             handleValidMethod,
	"/post":            handleValidMethod,
	"/put":             handleValidMethod,
	"/patch":           handleValidMethod,
	"/delete":          handleValidMethod,
	`/any(thing)?\b.*`: responses.InfoJSON,
}

func handleValidMethod(ex *exchange.Exchange) {
	allowedMethod := strings.ToUpper(strings.TrimPrefix(ex.URL.Path, "/"))
	if ex.Request.Method != allowedMethod {
		allowedMethods := allowedMethod + ", " + http.MethodOptions
		ex.ResponseWriter.Header().Set("Allow", allowedMethods)
		ex.ResponseWriter.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		if ex.Request.Method != http.MethodOptions {
			ex.ResponseWriter.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	responses.InfoJSON(ex)
}
