package server

import (
	"context"
	"github.com/sharat87/httpbun/bun"
	"github.com/sharat87/httpbun/server/spec"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type Server struct {
	httpServer *http.Server
	Addr       net.Addr
	closeCh    chan error
}

func StartNew(spec spec.Spec) Server {
	sslCertFile := os.Getenv("HTTPBUN_SSL_CERT")
	sslKeyFile := os.Getenv("HTTPBUN_SSL_KEY")

	server := &http.Server{
		Addr:    spec.BindTarget,
		Handler: bun.MakeBunHandler(spec),
	}

	listener, err := net.Listen("tcp", spec.BindTarget)
	if err != nil {
		log.Fatalf("Error listening on %q: %v", spec.BindTarget, err)
	}

	closeCh := make(chan error, 1)

	go func() {
		defer close(closeCh)
		if sslCertFile == "" {
			closeCh <- server.Serve(listener)
		} else {
			closeCh <- server.ServeTLS(listener, sslCertFile, sslKeyFile)
		}
	}()

	return Server{
		httpServer: server,
		Addr:       listener.Addr(),
		closeCh:    closeCh,
	}
}

func (s Server) Wait() error {
	return <-s.closeCh
}

func (s Server) CloseAndWait() {
	if s.httpServer != nil {
		ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelFunc()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down server: %v", err)
			if err := s.httpServer.Close(); err != nil {
				log.Printf("Error closing server: %v", err)
			}
		}
	}
	log.Print(s.Wait())
}
