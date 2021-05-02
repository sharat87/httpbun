package mux

import (
	"net/http"
	"regexp"
)

type Mux struct {
	Routes []route
}

type route struct {
	Pattern *regexp.Regexp
	Fn func(w http.ResponseWriter, req *http.Request, params map[string]string)
}

func New() *Mux {
	return &Mux{
		Routes: []route{},
	}
}

func (mux *Mux) HandleFunc(pattern string, fn func(w http.ResponseWriter, req *http.Request, params map[string]string)) {
	mux.Routes = append(mux.Routes, route{
		Pattern: regexp.MustCompile("^" + pattern + "$"),
		Fn: fn,
	})
}

func (mux Mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, route := range mux.Routes {
		match := route.Pattern.FindStringSubmatch(req.URL.Path)
		if match != nil {
			details := make(map[string]string)
			names := route.Pattern.SubexpNames()
			for i, name := range names {
				if name != "" {
					details[name] = match[i]
				}
			}
			route.Fn(w, req, details)
			return
		}
	}

	http.NotFound(w, req)
}
