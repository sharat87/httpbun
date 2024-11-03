package cache

import (
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/routes/responses"
	"net/http"
)

var Routes = map[string]exchange.HandlerFn2{
	"/cache":                handleCache,
	"/cache/(?P<age>\\d+)":  handleCacheControl,
	"/etag/(?P<etag>[^/]+)": handleEtag,
}

func handleCache(ex *exchange.Exchange) response.Response {
	shouldSendData :=
		ex.HeaderValueLast("If-Modified-Since") == "" &&
			ex.HeaderValueLast("If-None-Match") == ""

	if shouldSendData {
		return responses.InfoJSON(ex)
	} else {
		return response.New(http.StatusNotModified, nil, nil)
	}
}

func handleCacheControl(ex *exchange.Exchange) response.Response {
	// todo: setting header here is an abstraction leak
	ex.ResponseWriter.Header().Set("Cache-Control", "public, max-age="+ex.Field("age"))
	return responses.InfoJSON(ex)
}

func handleEtag(ex *exchange.Exchange) response.Response {
	// TODO: Handle If-Match header in etag endpoint: <https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Match>.
	etagInUrl := ex.Field("etag")
	etagInHeader := ex.HeaderValueLast("If-None-Match")

	if etagInUrl == etagInHeader {
		return response.New(http.StatusNotModified, nil, nil)
	} else {
		return responses.InfoJSON(ex)
	}
}
