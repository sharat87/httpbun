package server

import (
	"context"
	"errors"
	"fmt"
	"io"
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

	// A semaphore with a slot for each request being handled, up to the spec's limit.
	inFlight chan struct{}

	// A semaphore with a slot for each request that can take long, like `/delay`, up to the spec's limit. Handlers take
	// a slot with ex.HoldLongSlot.
	longInFlight chan struct{}
}

// How long to wait for a client to send a request's headers, so slow clients can't hold connections open.
const readHeaderTimeout = 10 * time.Second

// How long to keep an idle keep-alive connection open.
const idleTimeout = 2 * time.Minute

// No WriteTimeout or ReadTimeout, since endpoints like `/delay`, `/drip`, `/sse` and the `/llm` streams legitimately
// take minutes. Those stop when the client goes away instead.

// New creates a server for the given spec, without starting it. It's an http.Handler, so it can also be used directly,
// like with httptest.NewServer.
func New(spec spec.Spec) *Server {
	server := &Server{
		Server: &http.Server{
			ReadHeaderTimeout: readHeaderTimeout,
			IdleTimeout:       idleTimeout,
		},
		spec:         spec,
		closeCh:      make(chan error, 1),
		inFlight:     make(chan struct{}, spec.MaxConcurrentRequestsLimit()),
		longInFlight: make(chan struct{}, spec.MaxLongRequestsLimit()),
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

	// Health checks skip the limit, so a busy server isn't seen as a dead one, and restarted.
	if strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, s.spec.PathPrefix), "/") != "/health" {
		select {
		case s.inFlight <- struct{}{}:
			defer func() { <-s.inFlight }()
		default:
			// Don't queue, since queued requests would use up memory too.
			log.Printf("Busy, rejecting %s %s", req.Method, req.URL)
			w.Header().Set("Retry-After", response.RetryAfterSeconds)
			http.Error(w, "Server is busy, please try again later.", http.StatusServiceUnavailable)
			return
		}
	}

	maxBodySize := s.spec.MaxBodySizeLimit()
	if req.ContentLength > maxBodySize {
		bodyTooLarge(w, maxBodySize)
		return
	}
	body := &limitedBody{ReadCloser: http.MaxBytesReader(w, req.Body, maxBodySize)}
	req.Body = body

	ex := ex.New(w, req, s.spec)
	ex.SetLongSlots(s.longInFlight)
	// Handlers stop when the client goes away, so this also frees the slot then. It runs after Finish, which is where
	// streaming responses are written.
	defer ex.ReleaseLongSlot()

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

	var resp response.Response

	if s.spec.RootIsAny || s.spec.IsAnyHost(req.Host) {
		// Skip all route checking when root-is-any is enabled, or for any-hosts.
		info, err := responses.InfoJSON(ex)
		if err != nil {
			resp = response.BadRequest("%s", err.Error())
		} else {
			resp = response.Response{Body: info}
		}

	} else {
		found := false
		for _, route := range s.routes {
			if ex.MatchAndLoadFields(route.Pat) {
				resp = route.Fn(ex)
				found = true
				break
			}
		}
		if !found {
			log.Printf("NotFound ip=%s %s %s", incomingIP, req.Method, req.URL.String())
			http.NotFound(w, req)
			return
		}
	}

	// A handler that read a body without a Content-Length, past the limit, got an error, and made a response for that.
	// Whatever the error looked like to the handler, the client should know the body was too large.
	if body.tooLarge && resp.Status != http.StatusRequestEntityTooLarge {
		bodyTooLarge(w, maxBodySize)
		return
	}

	ex.Finish(resp)
}

func bodyTooLarge(w http.ResponseWriter, limit int64) {
	http.Error(w, fmt.Sprintf("Request body is too large, the limit is %d bytes.", limit), http.StatusRequestEntityTooLarge)
}

// limitedBody is a request body from http.MaxBytesReader, that remembers if a read failed for being over the limit.
type limitedBody struct {
	io.ReadCloser
	tooLarge bool
}

func (b *limitedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		b.tooLarge = true
	}
	return n, err
}
