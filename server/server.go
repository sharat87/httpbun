package server

import (
	"context"
	"github.com/sharat87/httpbun/bun"
	"github.com/sharat87/httpbun/info"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type Config struct {
	BindTarget string
	PathPrefix string
}

type Server struct {
	httpServer *http.Server
	Addr       net.Addr
	closeCh    chan error
}

func ParseArgs(args []string) Config {
	rc := &Config{}
	bindTarget := os.Getenv("HTTPBUN_BIND")

	i := 0

	for i < len(args) {
		arg := args[i]

		if arg == "--bind" {
			i++
			bindTarget = args[i]

		} else if arg == "--path-prefix" {
			i++
			rc.PathPrefix = args[i]

		} else {
			log.Fatalf("Unknown argument '%v'", arg)

		}

		i++
	}

	if bindTarget == "" {
		bindTarget = ":3090"
	}

	rc.BindTarget = bindTarget

	return *rc
}

func StartNew(config Config) Server {
	sslCertFile := os.Getenv("HTTPBUN_SSL_CERT")
	sslKeyFile := os.Getenv("HTTPBUN_SSL_KEY")

	m := bun.MakeBunHandler(config.PathPrefix, info.Commit, info.Date)

	server := &http.Server{
		Addr:    config.BindTarget,
		Handler: m,
	}

	listener, err := net.Listen("tcp", config.BindTarget)
	if err != nil {
		log.Fatalf("Error listening on %q: %v", config.BindTarget, err)
	}

	closeCh := make(chan error, 1)

	go func() {
		if sslCertFile == "" {
			closeCh <- server.Serve(listener)
		} else {
			closeCh <- server.ServeTLS(listener, sslCertFile, sslKeyFile)
		}
		close(closeCh)
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
