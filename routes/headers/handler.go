package headers

import (
	"fmt"
	"net/http"

	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/util"
)

var RouteList = []ex.Route{
	ex.NewRoute("/headers", handleHeaders),
	ex.NewRoute("/(response|respond-with)-headers?/?", handleResponseHeaders),
}

func handleHeaders(ex *ex.Exchange) response.Response {
	return response.Response{Body: map[string]any{"headers": ex.ExposableHeadersMap()}}
}

func handleResponseHeaders(ex *ex.Exchange) response.Response {
	responseHeaders := http.Header{}
	data := make(map[string]any)

	for name, values := range ex.Request.URL.Query() {
		name = http.CanonicalHeaderKey(name)
		for _, value := range values {
			responseHeaders.Add(name, value)
		}
		if len(values) > 1 {
			data[name] = values
		} else {
			data[name] = values[0]
		}
	}

	responseHeaders.Set(c.ContentType, c.ApplicationJSON)
	data[c.ContentType] = c.ApplicationJSON

	var jsonContent []byte

	for {
		jsonContent = util.ToJsonMust(map[string]any{"responseHeaders": data})
		newContentLength := fmt.Sprint(len(jsonContent))
		if data[c.ContentLength] == newContentLength {
			break
		}
		data[c.ContentLength] = newContentLength
	}

	return response.Response{
		Header: responseHeaders,
		Body:   jsonContent,
	}
}
