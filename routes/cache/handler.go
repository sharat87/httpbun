package cache

import (
	"net/http"

	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/routes/responses"
)

var Routes = map[string]exchange.HandlerFn{
	"/cache":                handleCache,
	"/cache/(?P<age>\\d+)":  handleCacheControl,
	"/etag/(?P<etag>[^/]+)": handleEtag,
}

func handleCache(ex *exchange.Exchange) response.Response {
	shouldSendData :=
		ex.HeaderValueLast("If-Modified-Since") == "" &&
			ex.HeaderValueLast("If-None-Match") == ""

	if shouldSendData {
		info, err := responses.InfoJSON(ex)
		if err != nil {
			return response.BadRequest(err.Error())
		}
		return response.Response{Body: info}
	} else {
		return response.Response{Status: http.StatusNotModified}
	}
}

func handleCacheControl(ex *exchange.Exchange) response.Response {
	res, err := responses.InfoJSON(ex)
	if err != nil {
		return response.BadRequest(err.Error())
	}

	return response.Response{
		Header: http.Header{
			"Cache-Control": {"public, max-age=" + ex.Field("age")},
		},
		Body: res,
	}
}

func handleEtag(ex *exchange.Exchange) response.Response {
	// TODO: Handle If-Match header in etag endpoint: <https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Match>.
	etagInUrl := ex.Field("etag")
	etagInHeader := ex.HeaderValueLast("If-None-Match")

	if etagInUrl == etagInHeader {
		return response.Response{Status: http.StatusNotModified}
	} else {
		info, err := responses.InfoJSON(ex)
		if err != nil {
			return response.BadRequest(err.Error())
		}
		return response.Response{Body: info}
	}
}
