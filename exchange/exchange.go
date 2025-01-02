package exchange

import (
	"fmt"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/server/spec"
	"github.com/sharat87/httpbun/util"
	"io"
	"log"
	"maps"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Exchange struct {
	Request        *http.Request
	responseWriter http.ResponseWriter
	fields         map[string]string // todo: this should be private!
	cappedBody     io.Reader
	RoutedPath     string
	ServerSpec     spec.Spec
}

type HandlerFn func(ex *Exchange) response.Response

func New(w http.ResponseWriter, req *http.Request, serverSpec spec.Spec) *Exchange {
	ex := &Exchange{
		Request:        req,
		responseWriter: w,
		fields:         map[string]string{},
		cappedBody:     io.LimitReader(req.Body, 10000),
		RoutedPath:     strings.TrimPrefix(req.URL.EscapedPath(), serverSpec.PathPrefix),
		ServerSpec:     serverSpec,
	}

	if req.URL.Host == "" && req.Host != "" {
		req.URL.Host = req.Host
	}

	if ex.responseWriter != nil {
		// todo: these common headers should be part of a "middleware" system

		// Need to set the exact origin, since `*` won't work if request includes credentials.
		// See <https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS/Errors/CORSNotSupportingCredentials>.
		originHeader := ex.HeaderValueLast("Origin")
		if originHeader != "" {
			ex.responseWriter.Header().Set("Access-Control-Allow-Origin", originHeader)
			ex.responseWriter.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		accessControlHeaders := ex.Request.Header.Get("Access-Control-Request-Headers")
		if accessControlHeaders != "" {
			ex.responseWriter.Header().Set("Access-Control-Allow-Headers", accessControlHeaders)
		}

		accessControlMethods := ex.Request.Header.Get("Access-Control-Request-Method")
		if accessControlMethods != "" {
			ex.responseWriter.Header().Set("Access-Control-Allow-Methods", accessControlMethods)
		}

		ex.responseWriter.Header().Set("X-Powered-By", "httpbun/"+serverSpec.Commit)
	}

	return ex
}

func (ex Exchange) MatchAndLoadFields(routePat string) bool {
	fields, isMatch := util.MatchRoutePat(routePat, ex.RoutedPath)
	if isMatch {
		maps.Copy(ex.fields, fields)
	}
	return isMatch
}

func (ex Exchange) Field(name string) string {
	return ex.fields[name]
}

func (ex Exchange) RedirectResponse(target string) *response.Response {
	if strings.HasPrefix(target, "/") {
		target = ex.ServerSpec.PathPrefix + target
	}

	return &response.Response{
		Status: http.StatusFound,
		Header: http.Header{
			"Location": {target},
		},
		Body: fmt.Sprintf(`<!doctype html>
<title>Redirecting...</title>
<h1>Redirecting...</h1>
<p>You should be redirected automatically to target URL: <a href=%q>%s</a>.  If not click the link.</p>
`, target, target),
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

	if len(ex.Request.TransferEncoding) > 0 {
		headers["Transfer-Encoding"] = ex.Request.TransferEncoding
	}

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
	if ex.cappedBody == nil {
		return nil
	}
	if bodyBytes, err := io.ReadAll(ex.cappedBody); err != nil {
		fmt.Println("Error reading request payload", err)
		return nil
	} else {
		return bodyBytes
	}
}

func (ex Exchange) BodyString() string {
	return string(ex.BodyBytes())
}

func (ex Exchange) Finish(resp response.Response) {
	if resp.Body != nil && resp.Writer != nil {
		ex.Finish(response.Response{
			Status: http.StatusInternalServerError,
			Body:   "Both Body and Writer are set in response. This isn't supported.",
		})
		return
	}

	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}

	maps.Copy(ex.responseWriter.Header(), resp.Header)

	for _, cookie := range resp.Cookies {
		ex.responseWriter.Header().Add("Set-Cookie", cookie.String())
	}

	if resp.Writer != nil {
		resp.Writer(response.NewBodyWriter(ex.responseWriter))
		return
	}

	var body []byte
	switch resp.Body.(type) {
	case nil:
		// do nothing
	case []byte:
		body = resp.Body.([]byte)
	case string:
		body = []byte(resp.Body.(string))
	default:
		ex.responseWriter.Header().Set("Content-Type", "application/json")
		body = util.ToJsonMust(resp.Body)
	}

	// Set `Content-Length` header, to disable chunked transfer. See https://github.com/sharat87/httpbun/issues/13
	ex.responseWriter.Header().Set("Content-Length", fmt.Sprint(len(body)))

	ex.responseWriter.WriteHeader(status)

	_, err := ex.responseWriter.Write(body)
	if err != nil {
		log.Printf("Error writing bytes to exchange response: %v\n", err)
	}
}
