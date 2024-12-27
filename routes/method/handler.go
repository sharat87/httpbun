package method

import (
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/routes/responses"
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

func handleValidMethod(ex *exchange.Exchange) response.Response {
	allowedMethod := strings.ToUpper(strings.TrimPrefix(ex.RoutedPath, "/"))
	if ex.Request.Method != allowedMethod {
		allowedMethods := allowedMethod + ", " + http.MethodOptions
		if ex.Request.Method != http.MethodOptions {
			return response.New(http.StatusMethodNotAllowed, http.Header{
				"Allow":                        []string{allowedMethods},
				"Access-Control-Allow-Methods": []string{allowedMethods},
			}, nil)
		}
	}

	return responses.InfoJSON(ex)
}
