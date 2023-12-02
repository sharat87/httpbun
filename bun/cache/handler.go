package cache

import (
	"github.com/sharat87/httpbun/bun/responses"
	"github.com/sharat87/httpbun/exchange"
	"net/http"
)

var Routes = map[string]exchange.HandlerFn{
	"/cache":                handleCache,
	"/cache/(?P<age>\\d+)":  handleCacheControl,
	"/etag/(?P<etag>[^/]+)": handleEtag,
}

func handleCache(ex *exchange.Exchange) {
	shouldSendData :=
		ex.HeaderValueLast("If-Modified-Since") == "" &&
			ex.HeaderValueLast("If-None-Match") == ""

	if shouldSendData {
		responses.InfoJSON(ex)
	} else {
		ex.ResponseWriter.WriteHeader(http.StatusNotModified)
	}
}

func handleCacheControl(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("Cache-Control", "public, max-age="+ex.Field("age"))
	responses.InfoJSON(ex)
}

func handleEtag(ex *exchange.Exchange) {
	// TODO: Handle If-Match header in etag endpoint: <https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Match>.
	etagInUrl := ex.Field("etag")
	etagInHeader := ex.HeaderValueLast("If-None-Match")

	if etagInUrl == etagInHeader {
		ex.ResponseWriter.WriteHeader(http.StatusNotModified)
	} else {
		responses.InfoJSON(ex)
	}
}
