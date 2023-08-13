package main

import (
	"github.com/sharat87/httpbun/server"
	"github.com/sharat87/httpbun/server/spec"
	"log"
	"os"
	"runtime"
)

func main() {
	c := spec.ParseArgs(os.Args[1:])
	log.Printf("Starting with %+v", c)

	log.Printf("OS: %q, Arch: %q.", runtime.GOOS, runtime.GOARCH)
	log.Printf("Commit: %q, Built: %q.", c.Commit, c.Date)

	s := server.StartNew(c)
	log.Printf("Serving on %s", s.Addr)
	log.Fatal(s.Wait())
}
