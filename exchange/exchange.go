package exchange

import (
	"fmt"
	"github.com/sharat87/httpbun/storage"
	"github.com/sharat87/httpbun/util"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Exchange struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	Fields         map[string]string
	CappedBody     io.Reader
	Origin         *string
	Storage        storage.Storage
	URL            *url.URL
}

func (ex Exchange) Field(name string) string {
	return ex.Fields[name]
}

func (ex Exchange) Redirect(w http.ResponseWriter, path string, changeToRelative bool) {
	if changeToRelative && strings.HasPrefix(path, "/") {
		path = strings.Repeat("../", strings.Count(ex.URL.Path, "/")-1) + strings.TrimPrefix(path, "/")
	}

	w.Header().Set("Location", path)
	w.WriteHeader(http.StatusFound)

	_, err := fmt.Fprintf(w, `<!doctype html>
<title>Redirecting...</title>
<h1>Redirecting...</h1>
<p>You should be redirected automatically to target URL: <a href=%q>%q</a>.  If not click the link.</p>
`, path, path)
	if err != nil {
		log.Printf("Error writing redirect HTML to HTTP response %v", err)
	}
}

func (ex Exchange) QueryParamInt(name string, value int) (int, error) {
	args := ex.Request.URL.Query()
	var err error

	if len(args[name]) > 0 {
		value, err = strconv.Atoi(args[name][0])
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", name)
		}
	}

	return value, nil
}

func (ex Exchange) QueryParamSingle(name string) (string, error) {
	return singleParamValue(ex.Request.URL.Query(), name)
}

func (ex Exchange) FormParamSingle(name string) (string, error) {
	return singleParamValue(ex.Request.Form, name)
}

func singleParamValue(args map[string][]string, name string) (string, error) {
	if len(args[name]) == 0 {
		return "", fmt.Errorf("missing required param %q", name)
	} else if len(args[name]) > 1 {
		return "", fmt.Errorf("too many values for param %q, expected only one", name)
	} else {
		return args[name][0], nil
	}
}

func (ex Exchange) HeaderValueLast(name string) string {
	if values := ex.Request.Header[name]; values != nil && len(values) > 0 {
		return values[len(values)-1]
	}

	return ""
}

func (ex Exchange) ExposableHeadersMap() map[string]string {
	headers := make(map[string]string)
	for name, values := range ex.Request.Header {
		headers[name] = strings.Join(values, ",")
	}
	return headers
}

func (ex Exchange) FindScheme() string {
	if os.Getenv("HTTPBUN_SSL_CERT") != "" || ex.HeaderValueLast("X-Forwarded-Proto") == "https" {
		return "https"
	}

	return "http"
}

func (ex Exchange) FullUrl() string {
	if !strings.HasPrefix(ex.Request.URL.String(), "/") {
		return ex.Request.URL.String()
	}

	return ex.FindScheme() + "://" + ex.Request.Host + ex.Request.URL.String()
}

// FindOrigin Find the IP address of the client that made this Exchange.
func (ex *Exchange) FindOrigin() string {
	if ex.Origin != nil {
		return *ex.Origin
	}

	// Compare with <http://httpbin.org/ip> or <http://checkip.amazonaws.com/> or <http://getmyip.co.in/>.
	ipStr := ""

	// The Forwarded header is a standard that Nginx can be configured to use.
	// Ref: <https://www.nginx.com/resources/wiki/start/topics/examples/forwarded/>.
	forwardedHeader := ex.HeaderValueLast("Forwarded")
	if forwardedHeader != "" {
		specs := util.ParseHeaderValueCsv(forwardedHeader)
		// Pick the last one among all `for` keys.
		for i := len(specs) - 1; i >= 0; i-- {
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
		ipStr = ex.HeaderValueLast("X-Forwarded-For")
	}

	// If that's also not available, get it directly from the connection.
	if ipStr == "" {
		if ip, _, err := net.SplitHostPort(ex.Request.RemoteAddr); err != nil {
			log.Printf("Unable to read IP from address %q.", ex.Request.RemoteAddr)
		} else if userIP := net.ParseIP(ip); userIP != nil {
			ipStr = userIP.String()
		}
	}

	ex.Origin = &ipStr
	return ipStr
}

func (ex Exchange) BodyString() string {
	if bodyBytes, err := ioutil.ReadAll(ex.CappedBody); err != nil {
		fmt.Println("Error reading request payload", err)
		return ""
	} else {
		return string(bodyBytes)
	}
}

func (ex Exchange) Write(content any) {
	_, err := fmt.Fprint(ex.ResponseWriter, content)
	if err != nil {
		log.Printf("Error writing to exchange response: %v\n", err)
	}
}

func (ex Exchange) WriteLn(content any) {
	ex.Write(content)
	ex.Write("\n")
}

func (ex Exchange) WriteF(content string, vars ...any) {
	ex.Write(fmt.Sprintf(content, vars...))
}

func (ex Exchange) RespondWithStatus(errorStatus int) {
	ex.ResponseWriter.WriteHeader(errorStatus)
	ex.WriteLn(http.StatusText(errorStatus))
}

func (ex Exchange) RespondError(status int, code, detail string) {
	ex.ResponseWriter.WriteHeader(status)
	util.WriteJson(ex.ResponseWriter, map[string]any{
		"error": map[string]any{
			"code":   code,
			"detail": detail,
		},
	})
}
