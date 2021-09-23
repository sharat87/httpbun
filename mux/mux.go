package mux

import (
	"fmt"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/storage"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type HandlerFn func(ex *exchange.Exchange)

type Mux struct {
	BeforeHandler HandlerFn
	Routes        []route
	Storage       storage.Storage
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

	url := &url.URL{
		Scheme:      req.URL.Scheme,
		Opaque:      req.URL.Opaque,
		User:        req.URL.User,
		Host:        req.URL.Host,
		Path:        req.URL.Path,
		RawPath:     req.URL.RawPath,
		ForceQuery:  req.URL.ForceQuery,
		RawQuery:    req.URL.RawQuery,
		Fragment:    req.URL.Fragment,
		RawFragment: req.URL.RawFragment,
	}

	ex := &exchange.Exchange{
		Request:        req,
		ResponseWriter: w,
		Fields:         make(map[string]string),
		CappedBody:     io.LimitReader(req.Body, 10000),
		Storage:        mux.Storage,
		URL:            url,
	}

	if ex.URL.Scheme == "" {
		// Other headers: <https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-Proto#examples>.
		if forwardedProto := ex.HeaderValueLast("X-Forwarded-Proto"); forwardedProto != "" {
			ex.URL.Scheme = forwardedProto
		} else if req.TLS == nil {
			ex.URL.Scheme = "http"
		} else {
			ex.URL.Scheme = "https"
		}
	}

	if ex.URL.Host == "" {
		if req.Host != "" {
			ex.URL.Host = req.Host
		} else if forwardedHost := ex.HeaderValueLast("X-Forwarded-Host"); forwardedHost != "" {
			ex.URL.Host = forwardedHost
		}
	}

	fmt.Printf("Processing URL %q %q.\n", ex.URL, req.URL)

	if ex.HeaderValueLast("X-Forwarded-Proto") == "http" && os.Getenv("HTTPBUN_FORCE_HTTPS") == "1" && ex.Request.URL.Path == "/" {
		ex.Redirect(w, "https://"+req.Host+req.URL.String())
		return
	}

	for _, route := range mux.Routes {
		match := route.Pattern.FindStringSubmatch(req.URL.Path)
		if match != nil {
			names := route.Pattern.SubexpNames()
			for i, name := range names {
				if name != "" {
					ex.Fields[name] = match[i]
				}
			}

			if mux.BeforeHandler != nil {
				mux.BeforeHandler(ex)
			}

			route.Fn(ex)
			return
		}
	}

	ip := ex.HeaderValueLast("X-Forwarded-For")
	log.Printf("NotFound ip=%s %s %s", ip, req.Method, req.URL.String())
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
