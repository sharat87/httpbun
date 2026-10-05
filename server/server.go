package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/routes"
	"github.com/sharat87/httpbun/routes/responses"
	"github.com/sharat87/httpbun/server/spec"
)

type Server struct {
	*http.Server
	spec    spec.Spec
	routes  []ex.Route
	closeCh chan error
}

// New creates a server for the given spec, without starting it. It's an http.Handler, so it can also be used directly,
// like with httptest.NewServer.
func New(spec spec.Spec) *Server {
	server := &Server{
		Server:  &http.Server{},
		spec:    spec,
		closeCh: make(chan error, 1),
	}
	server.Handler = server

	if !spec.RootIsAny {
		// When root is any, we don't need the route handlers at all.
		server.routes = routes.GetRoutes()
	}

	return server
}

func StartNew(spec spec.Spec) Server {
	bindTarget := spec.BindTarget
	if bindTarget == "" {
		if spec.TLSCertFile != "" {
			bindTarget = ":443"
		} else {
			bindTarget = ":80"
		}
	}

	server := New(spec)
	server.Addr = bindTarget

	listener, err := net.Listen("tcp", bindTarget)
	if err != nil {
		log.Fatalf("Error listening on %q: %v", spec.BindTarget, err)
	}

	go func() {
		defer close(server.closeCh)
		if spec.TLSCertFile == "" {
			server.closeCh <- server.Serve(listener)
		} else {
			server.closeCh <- server.ServeTLS(listener, spec.TLSCertFile, spec.TLSKeyFile)
		}
	}()

	return *server
}

func (s Server) Wait() error {
	return <-s.closeCh
}

func (s Server) CloseAndWait() {
	if s.Server != nil {
		ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelFunc()
		if err := s.Server.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down server: %v", err)
			if err := s.Server.Close(); err != nil {
				log.Printf("Error closing server: %v", err)
			}
		}
	}
	log.Print(s.Wait())
}

func (s Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !strings.HasPrefix(req.URL.Path, s.spec.PathPrefix) {
		http.NotFound(w, req)
		return
	}

	ex := ex.New(w, req, s.spec)

	// A bug in a handler shouldn't look like a network error to the client, so respond with a 500 that says what
	// went wrong. If the handler already started the response, this can only log.
	defer func() {
		if r := recover(); r != nil {
			if r == http.ErrAbortHandler {
				panic(r)
			}
			log.Printf("Panic handling %s %s: %v\n%s", req.Method, req.URL, r, debug.Stack())
			http.Error(w, fmt.Sprintf("Internal server error: %v", r), http.StatusInternalServerError)
		}
	}()

	incomingIP := ex.FindIncomingIPAddress()
	log.Printf(
		"From %s %s %s",
		incomingIP,
		req.Method,
		req.URL.String()[2:],
	)

	// Skip all route checking when root-is-any is enabled.
	if s.spec.RootIsAny {
		info, err := responses.InfoJSON(ex)
		if err != nil {
			ex.Finish(response.BadRequest("%s", err.Error()))
		} else {
			ex.Finish(response.Response{Body: info})
		}
		return
	}

	for _, route := range s.routes {
		if ex.MatchAndLoadFields(route.Pat) {
			ex.Finish(route.Fn(ex))
			return
		}
	}

	log.Printf("NotFound ip=%s %s %s", incomingIP, req.Method, req.URL.String())
	http.NotFound(w, req)
}
