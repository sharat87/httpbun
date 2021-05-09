package mux

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"github.com/sharat87/httpbun/request"
	"github.com/sharat87/httpbun/util"
)

type HandlerFn func(w http.ResponseWriter, req *request.Request)

type Mux struct {
	BeforeRequest HandlerFn
	Routes        []route
}

type route struct {
	Pattern *regexp.Regexp
	Fn      HandlerFn
}

func New() Mux {
	return Mux{
		Routes: []route{},
	}
}

func (mux *Mux) HandleFunc(pattern string, fn HandlerFn) {
	mux.Routes = append(mux.Routes, route{
		Pattern: regexp.MustCompile("^" + pattern + "$"),
		Fn:      fn,
	})
}

func (mux Mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	allowedHosts := strings.Split(os.Getenv("HTTPBUN_ALLOW_HOSTS"), ",")
	if !contains(allowedHosts, req.Host) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "%d Host %q not allowed", http.StatusForbidden, req.Host)
		return
	}

	req2 := &request.Request{
		Request: *req,
		Fields: make(map[string]string),
		CappedBody: io.LimitReader(req.Body, 10000),
	}

	if req2.URL.Path == "/" && util.HeaderValue(req2, "X-Forwarded-Proto") == "http" && os.Getenv("HTTPBUN_FORCE_HTTPS") == "true" {
		util.Redirect(w, req2, "https://" + req.Host + req.URL.String())
		return
	}

	for _, route := range mux.Routes {
		match := route.Pattern.FindStringSubmatch(req.URL.Path)
		if match != nil {
			names := route.Pattern.SubexpNames()
			for i, name := range names {
				if name != "" {
					req2.Fields[name] = match[i]
				}
			}

			if mux.BeforeRequest != nil {
				mux.BeforeRequest(w, req2)
			}

			route.Fn(w, req2)
			return
		}
	}

	log.Printf("NotFound %s %s", req.Method, req.URL.String())
	http.NotFound(w, req)
}

func contains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
