package headers

import (
	"fmt"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/util"
	"net/http"
)

var Routes = map[string]exchange.HandlerFn{
	"/headers":                            handleHeaders,
	"/(response|respond-with)-headers?/?": handleResponseHeaders,
}

func handleHeaders(ex *exchange.Exchange) {
	util.WriteJson(ex.ResponseWriter, ex.ExposableHeadersMap())
}

func handleResponseHeaders(ex *exchange.Exchange) {
	data := make(map[string]any)

	for name, values := range ex.Request.URL.Query() {
		name = http.CanonicalHeaderKey(name)
		for _, value := range values {
			ex.ResponseWriter.Header().Add(name, value)
		}
		if len(values) > 1 {
			data[name] = values
		} else {
			data[name] = values[0]
		}
	}

	ex.ResponseWriter.Header().Set(c.ContentType, c.ApplicationJSON)
	data[c.ContentType] = c.ApplicationJSON

	var jsonContent []byte

	for {
		jsonContent = util.ToJsonMust(data)
		newContentLength := fmt.Sprint(len(jsonContent))
		if data["Content-Length"] == newContentLength {
			break
		}
		data["Content-Length"] = newContentLength
	}

	ex.WriteBytes(jsonContent)
}
