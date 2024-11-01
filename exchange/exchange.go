package exchange

import (
	"fmt"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/server/spec"
	"github.com/sharat87/httpbun/util"
	"io"
	"log"
	"maps"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Exchange struct {
	Request         *http.Request
	ResponseWriter  http.ResponseWriter
	fields          map[string]string
	CappedBody      io.Reader
	RoutedPath      string
	OriginalPath    string
	RoutedRawPath   string
	OriginalRawPath string
	ServerSpec      spec.Spec
}

type HandlerFn func(ex *Exchange)

type HandlerFn2 func(ex *Exchange) response.Response

func New(w http.ResponseWriter, req *http.Request, serverSpec spec.Spec) *Exchange {
	ex := &Exchange{
		Request:         req,
		ResponseWriter:  w,
		fields:          map[string]string{},
		CappedBody:      io.LimitReader(req.Body, 10000),
		RoutedPath:      strings.TrimPrefix(req.URL.Path, serverSpec.PathPrefix), // deprecated
		OriginalPath:    req.URL.Path,
		RoutedRawPath:   strings.TrimPrefix(req.URL.EscapedPath(), serverSpec.PathPrefix),
		OriginalRawPath: req.URL.EscapedPath(),
		ServerSpec:      serverSpec,
	}

	if req.URL.Host == "" && req.Host != "" {
		req.URL.Host = req.Host
	}

	// Need to set the exact origin, since `*` won't work if request includes credentials.
	// See <https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS/Errors/CORSNotSupportingCredentials>.
	originHeader := ex.HeaderValueLast("Origin")
	if originHeader != "" {
		ex.ResponseWriter.Header().Set("Access-Control-Allow-Origin", originHeader)
		ex.ResponseWriter.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	accessControlHeaders := ex.Request.Header.Get("Access-Control-Request-Headers")
	if accessControlHeaders != "" {
		ex.ResponseWriter.Header().Set("Access-Control-Allow-Headers", accessControlHeaders)
	}

	accessControlMethods := ex.Request.Header.Get("Access-Control-Request-Method")
	if accessControlMethods != "" {
		ex.ResponseWriter.Header().Set("Access-Control-Allow-Methods", accessControlMethods)
	}

	ex.ResponseWriter.Header().Set("X-Powered-By", "httpbun/"+serverSpec.Commit)

	return ex
}

func (ex Exchange) MatchAndLoadFields(routePat regexp.Regexp) bool {
	match := routePat.FindStringSubmatch(ex.RoutedRawPath)
	if match != nil {
		names := routePat.SubexpNames()
		for i, name := range names {
			if name != "" {
				ex.fields[name] = match[i]
			}
		}
		return true
	}
	return false
}

func (ex Exchange) Field(name string) string {
	return ex.fields[name]
}

func (ex Exchange) Redirect(target string) {
	if strings.HasPrefix(target, "/") {
		target = ex.ServerSpec.PathPrefix + target
	}

	ex.ResponseWriter.Header().Set("Location", target)
	ex.ResponseWriter.WriteHeader(http.StatusFound)

	_, err := fmt.Fprintf(ex.ResponseWriter, `<!doctype html>
<title>Redirecting...</title>
<h1>Redirecting...</h1>
<p>You should be redirected automatically to target URL: <a href=%q>%s</a>.  If not click the link.</p>
`, target, target)
	if err != nil {
		log.Printf("Error writing redirect HTML to HTTP response %v", err)
	}
}

func (ex Exchange) QueryParamInt(name string, value int) (int, error) {
	args := ex.Request.URL.Query()

	if len(args[name]) > 0 {
		var err error
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

func (ex Exchange) ExposableHeadersMap() map[string]any {
	headers := map[string]any{}

	for name, values := range ex.Request.Header {
		if !strings.HasPrefix(name, "X-Httpbun-") {
			if len(values) > 1 {
				headers[name] = values
			} else {
				headers[name] = values[0]
			}
		}
	}

	return headers
}

func (ex Exchange) FindScheme() string {
	if forwardedProto := ex.HeaderValueLast("X-Httpbun-Forwarded-Proto"); forwardedProto != "" {
		return forwardedProto
	}

	// todo: this should use the current server's spec, not the global env var to decide if TLS is enabled
	if os.Getenv("HTTPBUN_TLS_CERT") != "" {
		return "https"
	}

	return "http"
}

func (ex Exchange) FullUrl() string {
	if !strings.HasPrefix(ex.Request.URL.String(), "/") {
		return ex.Request.URL.String()
	}

	return ex.FindScheme() + ":" + ex.Request.URL.String()
}

// FindIncomingIPAddress Find the IP address of the client that made this Exchange.
func (ex Exchange) FindIncomingIPAddress() string {
	// Compare with <http://httpbin.org/ip> or <http://checkip.amazonaws.com/> or <http://getmyip.co.in/>.
	ipStr := ex.HeaderValueLast("X-Httpbun-Forwarded-For")

	// If that's also not available, get it directly from the connection.
	if ipStr == "" {
		if ip, _, err := net.SplitHostPort(ex.Request.RemoteAddr); err != nil {
			log.Printf("Unable to read IP from address %q.", ex.Request.RemoteAddr)
		} else if userIP := net.ParseIP(ip); userIP != nil {
			ipStr = userIP.String()
		}
	}

	return ipStr
}

func (ex Exchange) BodyBytes() []byte {
	if bodyBytes, err := io.ReadAll(ex.CappedBody); err != nil {
		fmt.Println("Error reading request payload", err)
		return nil
	} else {
		return bodyBytes
	}
}

func (ex Exchange) BodyString() string {
	return string(ex.BodyBytes())
}

func (ex Exchange) Write(content string) {
	_, err := fmt.Fprint(ex.ResponseWriter, content)
	if err != nil {
		log.Printf("Error writing to exchange response: %v\n", err)
	}
}

func (ex Exchange) WriteLn(content string) {
	ex.Write(content)
	ex.Write("\n")
}

func (ex Exchange) WriteBytes(content []byte) {
	_, err := ex.ResponseWriter.Write(content)
	if err != nil {
		log.Printf("Error writing bytes to exchange response: %v\n", err)
	}
}

func (ex Exchange) WriteF(content string, vars ...any) {
	ex.Write(fmt.Sprintf(content, vars...))
}

func (ex Exchange) WriteJSON(data any) {
	w := ex.ResponseWriter
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(util.ToJsonMust(data))
	if err != nil {
		log.Printf("Error writing JSON to HTTP response %v", err)
	}
}

func (ex Exchange) RespondWithStatus(errorStatus int) {
	ex.Finish(response.New(errorStatus, nil, []byte(http.StatusText(errorStatus)+"\n")))
}

func (ex Exchange) RespondBadRequest(message string, vars ...any) {
	ex.Finish(response.BadRequest(message, vars...))
}

func (ex Exchange) RespondError(status int, code, detail string) {
	ex.Finish(response.New(
		status,
		http.Header{
			c.ContentType: []string{c.ApplicationJSON},
		},
		util.ToJsonMust(map[string]any{
			"error": map[string]any{
				"code":   code,
				"detail": detail,
			},
		}),
	))
}

func (ex Exchange) Finish(resp response.Response) {
	maps.Copy(ex.ResponseWriter.Header(), resp.Header)
	ex.ResponseWriter.WriteHeader(resp.Status)
	ex.WriteBytes(resp.Body)
}
