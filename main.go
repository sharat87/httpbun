package main

import (
	"github.com/sharat87/httpbun/server"
	"github.com/sharat87/httpbun/server/spec"
	"log"
	"runtime"
)

func main() {
	c := spec.ParseArgs()
	log.Printf("Starting with %+v", c)

	log.Printf("OS: %q, Arch: %q.", runtime.GOOS, runtime.GOARCH)
	log.Printf("Commit: %q, Built: %q.", c.Commit, c.Date)

	s := server.StartNew(c)
	log.Printf("Serving on %v", s.Addr)
	log.Fatal(s.Wait())
}
