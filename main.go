package main

import (
	"github.com/sharat87/httpbun/request"
	"github.com/sharat87/httpbun/bun"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"
)

var (
	Version string
	Commit  string
	Date    string
)

func main() {
	rand.Seed(time.Now().Unix())

	host, ok := os.LookupEnv("HOST")
	if !ok {
		host = "localhost"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3090"
	}

	// Genreate a self-signed cert with following command:
	// openssl req -x509 -newkey rsa:4096 -nodes -out cert.pem -keyout key.pem -days 365 -subj "/O=httpbun/CN=httpbun.com"
	sslCertFile := os.Getenv("HTTPBUN_SSL_CERT")
	sslKeyFile := os.Getenv("HTTPBUN_SSL_KEY")

	m := bun.MakeBunHandler()
	m.BeforeRequest = func(w http.ResponseWriter, req *request.Request) {
		ip := req.HeaderValueLast("X-Forwarded-For")
		log.Printf("Handling ip=%s %s %s%s", ip, req.Method, req.Host, req.URL.String())
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		poweredBy := "httpbun"
		if Version != "" {
			poweredBy += " " + Version
		}
		w.Header().Set("X-Powered-By", poweredBy)
	}

	s := &http.Server{
		Addr:    host + ":" + port,
		Handler: m,
	}

	scheme := "http"
	if sslCertFile != "" {
		scheme = "https"
	}

	log.Printf("Serving on %s://%s:%s (set HOST / PORT environment variables to change)...\n", scheme, host, port)
	log.Printf("Version: %q, Commit: %q, Date: %q.\n", Version, Commit, Date)
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "'Error getting hostname: " + err.Error() + "'"
	}
	log.Printf("OS: %q, Arch: %q, Host: %q.\n", runtime.GOOS, runtime.GOARCH, hostname)

	if sslCertFile == "" {
		log.Fatal(s.ListenAndServe())
	} else {
		log.Fatal(s.ListenAndServeTLS(sslCertFile, sslKeyFile))
	}
}
