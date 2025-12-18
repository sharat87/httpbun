package ex

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/server/spec"
	"github.com/sharat87/httpbun/util"
)

func InvokeHandlerForTest(path string, req http.Request, routePat string, fn HandlerFn) response.Response {
	if req.URL != nil {
		panic("req.URL must be nil")
	}
	// Prepend a `/` to the path to ensure `req.URL.Path` is consistent inside the handler. Otherwise, the hash
	// computation in digest auth will fail, since it depends on the URL path.
	req.URL, _ = url.Parse("http://localhost/" + path)

	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		if req.Header == nil {
			req.Header = http.Header{}
		}
		req.Header.Set("Content-Length", strconv.Itoa(len(bodyBytes)))
	}

	ex := New(
		nil,
		&req,
		spec.Spec{
			PathPrefix: "",
		},
	)

	var isMatch bool
	ex.fields, isMatch = util.MatchRoutePat(MakePat(routePat), ex.RoutedPath)
	if !isMatch {
		panic("Route pattern did not match path")
	}

	return fn(ex)
}
