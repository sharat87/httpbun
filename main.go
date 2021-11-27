package main

import (
	"github.com/sharat87/httpbun/bun"
	"github.com/sharat87/httpbun/exchange"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

var (
	Version string
	Commit  string
	Date    string
)

func main() {
	rand.Seed(time.Now().Unix())

	protocol := "tcp"
	bind_target, ok := os.LookupEnv("BIND")
	if !ok {
		bind_target = "localhost:3090"
	}

	if strings.HasPrefix(bind_target, "unix/") {
		protocol = "unix"
		bind_target = strings.TrimPrefix(bind_target, "unix/")
	}

	listener, err := net.Listen(protocol, bind_target)
	if err != nil {
		log.Fatal("Error creating listener.", err)
	}

	// Genreate a self-signed cert with following command:
	// openssl req -x509 -newkey rsa:4096 -nodes -out cert.pem -keyout key.pem -days 365 -subj "/O=httpbun/CN=httpbun.com"
	sslCertFile := os.Getenv("HTTPBUN_SSL_CERT")
	sslKeyFile := os.Getenv("HTTPBUN_SSL_KEY")

	m := bun.MakeBunHandler()
	m.BeforeHandler = func(ex *exchange.Exchange) {
		ip := ex.HeaderValueLast("X-Forwarded-For")
		log.Printf("Handling ip=%s %s %s%s", ip, ex.Request.Method, ex.Request.Host, ex.Request.URL.String())

		// Need to set the exact origin, since `*` won't work if request includes credentials.
		// See <https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS/Errors/CORSNotSupportingCredentials>.
		originHeader := ex.HeaderValueLast("Origin")
		if originHeader != "" {
			ex.ResponseWriter.Header().Set("Access-Control-Allow-Origin", originHeader)
			ex.ResponseWriter.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		poweredBy := "httpbun"
		if Version != "" {
			poweredBy += " " + Version
		}
		ex.ResponseWriter.Header().Set("X-Powered-By", poweredBy)
	}

	scheme := "http"
	if sslCertFile != "" {
		scheme = "https"
	}

	log.Printf("Version: %q, Commit: %q, Built: %q.\n", Version, Commit, Date)

	// To get port being used as an int: listener.Addr().(*net.TCPAddr).Port
	log.Printf(
		"Serving on %s://%s (set HOST / PORT environment variables to change)...\n",
		scheme,
		listener.Addr(),
	)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "'Error getting hostname: " + err.Error() + "'"
	}
	log.Printf("OS: %q, Arch: %q, Host: %q.\n", runtime.GOOS, runtime.GOARCH, hostname)

	if sslCertFile == "" {
		log.Fatal(http.Serve(listener, m))
	} else {
		log.Fatal(http.ServeTLS(listener, m, sslCertFile, sslKeyFile))
	}
}
