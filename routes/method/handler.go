package method

import (
	"net/http"
	"strings"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/routes/responses"
)

var RouteList = []ex.Route{
	ex.NewRoute("/get", handleValidMethod),
	ex.NewRoute("/post", handleValidMethod),
	ex.NewRoute("/put", handleValidMethod),
	ex.NewRoute("/patch", handleValidMethod),
	ex.NewRoute("/delete", handleValidMethod),
	ex.NewRoute(`/any(thing)?\b.*`, handleAnything),
}

func handleAnything(ex *ex.Exchange) response.Response {
	info, err := responses.InfoJSON(ex)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}
	return response.Response{Body: info}
}

func handleValidMethod(ex *ex.Exchange) response.Response {
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
		return response.BadRequest("%s", err.Error())
	}
	return response.Response{Body: info}
}
