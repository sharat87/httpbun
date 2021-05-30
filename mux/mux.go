package mux

import (
	"fmt"
	"github.com/sharat87/httpbun/request"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type HandlerFn func(w http.ResponseWriter, req *request.Request)

type Mux struct {
	BeforeHandler HandlerFn
	Routes        []route
}

type route struct {
	Pattern regexp.Regexp
	Fn      HandlerFn
}

func (mux *Mux) HandleFunc(pattern string, fn HandlerFn) {
	mux.Routes = append(mux.Routes, route{
		Pattern: *regexp.MustCompile("^" + pattern + "$"),
		Fn:      fn,
	})
}

func (mux Mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// TODO: Don't parse HTTPBUN_ALLOW_HOSTS on every request.
	allowedHostsStr := os.Getenv("HTTPBUN_ALLOW_HOSTS")
	if allowedHostsStr != "" {
		allowedHosts := strings.Split(allowedHostsStr, ",")
		if !contains(allowedHosts, req.Host) {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "%d Host %q not allowed", http.StatusForbidden, req.Host)
			return
		}
	}

	req2 := &request.Request{
		Request:    *req,
		Fields:     make(map[string]string),
		CappedBody: io.LimitReader(req.Body, 10000),
	}

	if req2.HeaderValueLast("X-Forwarded-Proto") == "http" && os.Getenv("HTTPBUN_FORCE_HTTPS") == "1" {
		if req2.URL.Path == "/" {
			req2.Redirect(w, "https://"+req.Host+req.URL.String())
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Please use https")
		}
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

			if mux.BeforeHandler != nil {
				mux.BeforeHandler(w, req2)
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
