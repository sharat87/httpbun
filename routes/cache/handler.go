package cache

import (
	"net/http"
	"strings"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/routes/responses"
)

var EtagRoute = "/etag/(?P<etag>[^/]+)"

var RouteList = []ex.Route{
	ex.NewRoute("/cache", handleCache),
	ex.NewRoute("/cache/(?P<age>\\d+)", handleCacheControl),
	ex.NewRoute(EtagRoute, handleEtag),
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
	if !validEtagOpaqueValue(etagInUrl) {
		return response.BadRequest("Invalid ETag value")
	}
	etag := `"` + etagInUrl + `"`
	header := make(http.Header)
	header.Set("ETag", etag)

	if matchesIfNoneMatch(ex.Request.Header.Values("If-None-Match"), etag) {
		status := http.StatusPreconditionFailed
		if ex.Request.Method == http.MethodGet || ex.Request.Method == http.MethodHead {
			status = http.StatusNotModified
		}
		return response.Response{Status: status, Header: header}
	}

	info, err := responses.InfoJSON(ex)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}
	return response.Response{Header: header, Body: info}
}

func validEtagOpaqueValue(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x21 || value[i] == '"' || value[i] == 0x7f {
			return false
		}
	}
	return true
}

func matchesIfNoneMatch(values []string, etag string) bool {
	// Keep the single bare validator accepted by earlier versions.
	if len(values) == 1 && values[0] == etag[1:len(etag)-1] {
		return true
	}
	value := strings.Trim(strings.Join(values, ","), " \t")
	if value == "*" {
		return true
	}

	matched := false
	for value != "" {
		value = strings.TrimLeft(value, " \t")
		if value == "" {
			break
		}
		if value[0] == ',' {
			value = value[1:]
			continue
		}
		value = strings.TrimPrefix(value, "W/")
		if len(value) == 0 || value[0] != '"' {
			return false
		}
		end := strings.IndexByte(value[1:], '"')
		if end < 0 {
			return false
		}
		tag := value[:end+2]
		if !validEtagOpaqueValue(tag[1 : len(tag)-1]) {
			return false
		}
		value = strings.TrimLeft(value[end+2:], " \t")
		if value != "" && value[0] != ',' {
			return false
		}
		matched = matched || tag == etag
	}
	return matched
}
