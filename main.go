package main

import (
	"github.com/sharat87/httpbun/info"
	"github.com/sharat87/httpbun/server"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"
)

func main() {
	config := server.ParseArgs(os.Args[1:])
	log.Printf("Starting with %+v", config)

	rand.Seed(time.Now().Unix())

	log.Printf("OS: %q, Arch: %q.", runtime.GOOS, runtime.GOARCH)
	log.Printf("Commit: %q, Built: %q.", info.Commit, info.Date)

	s := server.StartNew(config)
	log.Printf("Serving on %s", s.Addr)
	log.Fatal(s.Wait())
}
