package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
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

func StartNew(spec spec.Spec) Server {
	tlsCertFile := os.Getenv("HTTPBUN_TLS_CERT")
	tlsKeyFile := os.Getenv("HTTPBUN_TLS_KEY")

	bindTarget := spec.BindTarget
	if bindTarget == "" {
		if tlsCertFile != "" {
			bindTarget = ":443"
		} else {
			bindTarget = ":80"
		}
	}

	server := &Server{
		Server: &http.Server{
			Addr: bindTarget,
		},
		spec:    spec,
		closeCh: make(chan error, 1),
	}
	server.Handler = server

	if !spec.RootIsAny {
		// When root is any, we don't need the route handlers at all.
		server.routes = routes.GetRoutes()
	}

	listener, err := net.Listen("tcp", bindTarget)
	if err != nil {
		log.Fatalf("Error listening on %q: %v", spec.BindTarget, err)
	}

	go func() {
		defer close(server.closeCh)
		if tlsCertFile == "" {
			server.closeCh <- server.Serve(listener)
		} else {
			server.closeCh <- server.ServeTLS(listener, tlsCertFile, tlsKeyFile)
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
