package method

import (
	"net/http"
	"strings"

	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/routes/responses"
)

var Routes = map[string]exchange.HandlerFn{
	"/get":             handleValidMethod,
	"/post":            handleValidMethod,
	"/put":             handleValidMethod,
	"/patch":           handleValidMethod,
	"/delete":          handleValidMethod,
	`/any(thing)?\b.*`: handleAnything,
}

func handleAnything(ex *exchange.Exchange) response.Response {
	info, err := responses.InfoJSON(ex)
	if err != nil {
		return response.BadRequest(err.Error())
	}
	return response.Response{Body: info}
}

func handleValidMethod(ex *exchange.Exchange) response.Response {
	allowedMethod := strings.ToUpper(strings.TrimPrefix(ex.RoutedPath, "/"))
	if ex.Request.Method != allowedMethod {
		allowedMethods := allowedMethod + ", " + http.MethodOptions
		if ex.Request.Method != http.MethodOptions {
			return response.Response{
				Status: http.StatusMethodNotAllowed,
				Header: http.Header{
					"Allow":                        []string{allowedMethods},
					"Access-Control-Allow-Methods": []string{allowedMethods},
				},
			}
		}
	}

	info, err := responses.InfoJSON(ex)
	if err != nil {
		return response.BadRequest(err.Error())
	}
	return response.Response{Body: info}
}
