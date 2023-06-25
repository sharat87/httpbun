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

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "'Error getting hostname: " + err.Error() + "'"
	}
	log.Printf("OS: %q, Arch: %q, Host: %q.", runtime.GOOS, runtime.GOARCH, hostname)
	log.Printf("Version: %q, Commit: %q, Built: %q.", info.Version, info.Commit, info.Date)

	s := server.StartNew(config)
	log.Printf("Serving on %s", s.Addr)
	log.Fatal(s.Wait())
}
