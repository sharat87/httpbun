package mux

import (
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/server/spec"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type HandlerFn func(ex *exchange.Exchange)

type Mux struct {
	routes     []route
	ServerSpec spec.Spec
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
	if !strings.HasPrefix(req.URL.Path, mux.ServerSpec.PathPrefix) {
		http.NotFound(w, req)
		return
	}

	ex := exchange.New(w, req, mux.ServerSpec)

	incomingIP := ex.FindIncomingIPAddress()
	log.Printf(
		"From ip=%s %s %s%s",
		incomingIP,
		req.Method,
		req.Host,
		req.URL.String(),
	)

	for _, route := range mux.routes {
		if ex.MatchAndLoadFields(route.pat) {
			route.fn(ex)
			return
		}
	}

	log.Printf("NotFound ip=%s %s %s", incomingIP, req.Method, req.URL.String())
	http.NotFound(w, req)
}
