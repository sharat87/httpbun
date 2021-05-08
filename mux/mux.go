package mux

import (
	"io"
	"log"
	"net/http"
	"regexp"
)

type HandlerFn func(w http.ResponseWriter, req *Request)

type Mux struct {
	BeforeRequest HandlerFn
	Routes        []route
}

type route struct {
	Pattern *regexp.Regexp
	Fn      HandlerFn
}

type Request struct {
	http.Request
	fields map[string]string
	CappedBody io.Reader
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
	for _, route := range mux.Routes {
		match := route.Pattern.FindStringSubmatch(req.URL.Path)
		if match != nil {
			req2 := &Request{
				*req,
				make(map[string]string),
				io.LimitReader(req.Body, 10000),
			}

			names := route.Pattern.SubexpNames()
			for i, name := range names {
				if name != "" {
					req2.fields[name] = match[i]
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

func (req Request) Field(name string) string {
	return req.fields[name]
}
