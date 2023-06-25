package mux

import (
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/info"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type HandlerFn func(ex *exchange.Exchange)

type Mux struct {
	PathPrefix string
	routes     []route
}

type route struct {
	pat regexp.Regexp
	fn  HandlerFn
}

func (mux *Mux) HandleFunc(pattern string, fn HandlerFn) {
	mux.routes = append(mux.routes, route{
		pat: *regexp.MustCompile("^" + pattern + "$"),
		fn:  fn,
	})
}

func (mux Mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !strings.HasPrefix(req.URL.Path, mux.PathPrefix) {
		http.NotFound(w, req)
		return
	}

	ex := &exchange.Exchange{
		Request:        req,
		ResponseWriter: w,
		Fields:         make(map[string]string),
		CappedBody:     io.LimitReader(req.Body, 10000),
		URL: &url.URL{
			Scheme:      req.URL.Scheme,
			Opaque:      req.URL.Opaque,
			User:        req.URL.User,
			Host:        req.URL.Host,
			Path:        strings.TrimPrefix(req.URL.Path, mux.PathPrefix),
			RawPath:     req.URL.RawPath,
			ForceQuery:  req.URL.ForceQuery,
			RawQuery:    req.URL.RawQuery,
			Fragment:    req.URL.Fragment,
			RawFragment: req.URL.RawFragment,
		},
	}

	if ex.URL.Host == "" && req.Host != "" {
		ex.URL.Host = req.Host
	}

	incomingIP := ex.FindIncomingIPAddress()
	log.Printf(
		"From ip=%s %s %s%s",
		incomingIP,
		req.Method,
		req.Host,
		req.URL.String(),
	)

	// Need to set the exact origin, since `*` won't work if request includes credentials.
	// See <https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS/Errors/CORSNotSupportingCredentials>.
	originHeader := ex.HeaderValueLast("Origin")
	if originHeader != "" {
		ex.ResponseWriter.Header().Set("Access-Control-Allow-Origin", originHeader)
		ex.ResponseWriter.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	ex.ResponseWriter.Header().Set("X-Powered-By", "httpbun/"+info.Commit)

	for _, route := range mux.routes {
		match := route.pat.FindStringSubmatch(ex.URL.Path)
		if match != nil {
			names := route.pat.SubexpNames()
			for i, name := range names {
				if name != "" {
					ex.Fields[name] = match[i]
				}
			}

			route.fn(ex)
			return
		}
	}

	log.Printf("NotFound ip=%s %s %s", incomingIP, req.Method, req.URL.String())
	http.NotFound(w, req)
}
