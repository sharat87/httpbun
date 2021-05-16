package request

import (
	"log"
	"net"
	"github.com/sharat87/httpbun/util"
	"strings"
	"fmt"
	"strconv"
	"io"
	"net/http"
	"os"
)

var hiddenHeaders = map[string]bool{
	"Total-Route-Time":  true,
	"Via":               true,
	"X-Forwarded-For":   true,
	"X-Forwarded-Port":  true,
	"X-Forwarded-Proto": true,
	"X-Request-Id":      true,
	"X-Request-Start":   true,
}

type Request struct {
	http.Request
	Fields     map[string]string
	CappedBody io.Reader
	Origin *string
}

func (req Request) Field(name string) string {
	return req.Fields[name]
}

func (req Request) Redirect(w http.ResponseWriter, path string) {
	if strings.HasPrefix(path, "/") {
		path = strings.Repeat("../", strings.Count(req.URL.Path, "/")-1) + strings.TrimPrefix(path, "/")
	}

	w.Header().Set("Location", path)
	w.WriteHeader(http.StatusFound)

	fmt.Fprintf(w, `<!doctype html>
<title>Redirecting...</title>
<h1>Redirecting...</h1>
<p>You should be redirected automatically to target URL: <a href=%q>/cookies</a>.  If not click the link.</p>`, path)
}

func (req Request) QueryParamInt(name string, value int) (int, error) {
	args := req.URL.Query()
	var err error

	if len(args[name]) > 0 {
		value, err = strconv.Atoi(args[name][0])
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", name)
		}
	}

	return value, nil
}

func (req Request) HeaderValueLast(name string) string {
	if values := req.Header[name]; values != nil && len(values) > 0 {
		return values[len(values)-1]
	}

	return ""
}

func (req Request) ExposableHeadersMap() map[string]string {
	headers := make(map[string]string)
	for name, values := range req.Header {
		if !hiddenHeaders[name] {
			headers[name] = strings.Join(values, ",")
		}
	}
	return headers
}

func (req Request) FullUrl() string {
	if !strings.HasPrefix(req.URL.String(), "/") {
		return req.URL.String()
	}

	scheme := "http"
	if os.Getenv("HTTPBUN_SSL_CERT") != "" || req.HeaderValueLast("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	return scheme + "://" + req.Host + req.URL.String()
}

func (req *Request) FindOrigin() string {
	if req.Origin != nil {
		return *req.Origin
	}

	// Compare with <http://httpbin.org/ip> or <http://checkip.amazonaws.com/> or <http://getmyip.co.in/>.
	ipStr := ""

	// The Forwarded header is a standard that Nginx can be configured to use.
	// Ref: <https://www.nginx.com/resources/wiki/start/topics/examples/forwarded/>.
	forwardedHeader := req.HeaderValueLast("Forwarded")
	if forwardedHeader != "" {
		specs := util.ParseHeaderValueCsv(forwardedHeader)
		// Pick the last one among all `for` keys.
		for i := len(specs)-1; i >= 0; i-- {
			ipStr = specs[i]["for"]
			if ipStr != "" {
				break
			}
		}
	}

	// Get it from Nginx's `$proxy_add_x_forwarded_for` based configuration.
	// Heroku also sends the actual IP in the `X-Forwarded-For` header:
	// <https://devcenter.heroku.com/articles/http-routing#heroku-headers>
	// AWS' ALBs also use the same header:
	// <https://docs.aws.amazon.com/elasticloadbalancing/latest/userguide/how-elastic-load-balancing-works.html#http-headers>
	if ipStr == "" {
		ipStr = req.HeaderValueLast("X-Forwarded-For")
	}

	// If that's also not available, get it directly from the connection.
	if ipStr == "" {
		if ip, _, err := net.SplitHostPort(req.RemoteAddr); err != nil {
			log.Printf("Unable to read IP from address %q.", req.RemoteAddr)
		} else if userIP := net.ParseIP(ip); userIP != nil {
			ipStr = userIP.String()
		}
	}

	req.Origin = &ipStr
	return ipStr
}
