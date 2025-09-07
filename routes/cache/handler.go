package cache

import (
	"net/http"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/routes/responses"
)

var RouteList = []ex.Route{
	ex.NewRoute("/cache", handleCache),
	ex.NewRoute("/cache/(?P<age>\\d+)", handleCacheControl),
	ex.NewRoute("/etag/(?P<etag>[^/]+)", handleEtag),
}

func handleCache(ex *ex.Exchange) response.Response {
	shouldSendData :=
		ex.HeaderValueLast("If-Modified-Since") == "" &&
			ex.HeaderValueLast("If-None-Match") == ""

	if shouldSendData {
		info, err := responses.InfoJSON(ex)
		if err != nil {
			return response.BadRequest("%s", err.Error())
		}
		return response.Response{Body: info}
	} else {
		return response.Response{Status: http.StatusNotModified}
	}
}

func handleCacheControl(ex *ex.Exchange) response.Response {
	res, err := responses.InfoJSON(ex)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	return response.Response{
		Header: http.Header{
			"Cache-Control": {"public, max-age=" + ex.Field("age")},
		},
		Body: res,
	}
}

func handleEtag(ex *ex.Exchange) response.Response {
	// TODO: Handle If-Match header in etag endpoint: <https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Match>.
	etagInUrl := ex.Field("etag")
	etagInHeader := ex.HeaderValueLast("If-None-Match")

	if etagInUrl == etagInHeader {
		return response.Response{Status: http.StatusNotModified}
	} else {
		info, err := responses.InfoJSON(ex)
		if err != nil {
			return response.BadRequest("%s", err.Error())
		}
		return response.Response{Body: info}
	}
}
