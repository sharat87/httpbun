package spec

import (
	"flag"
	"net"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/sharat87/httpbun/util"
)

var (
	Commit string
	Date   string
)

type Spec struct {
	BindTarget string
	PathPrefix string

	// If true, no route handlers are registered on any path, and `/` behaves like `/any`. This means that none of the
	// UI pages will be accessible either. Like, opening `/` to see the homepage won't work.
	RootIsAny bool

	Commit      string
	CommitShort string
	Date        string

	// Route configurations
	EndpointBytesSizeLimit int

	// TLS is enabled when both of these are set.
	TLSCertFile string
	TLSKeyFile  string

	// Hosts that absolute redirect targets are allowed to point to. Entries like `*.example.com` allow subdomains.
	// When nil, DefaultAllowedRedirectDomains is used.
	AllowedRedirectDomains []string

	// Env variables exposed by the `/info` endpoint.
	InfoEnv map[string]string

	// Headers that a proxy in front of httpbun sets, with the client's IP and the scheme it used. When empty,
	// DefaultClientIPHeader and DefaultSchemeHeader are used. These are trusted as is, so they should only be changed
	// when the proxy always overwrites these headers.
	ClientIPHeader string
	SchemeHeader   string

	// Request headers left out of responses that echo them, like `/headers`. Entries can have `*` wildcards, like
	// `X-Forwarded-*`. The client IP and scheme headers, and all `X-Httpbun-*` headers, are always left out.
	HideHeaders []string

	// Hosts on which all endpoints behave like `/any`, like RootIsAny, but only for requests to these hosts.
	AnyHosts []string

	// If true, HTML pages served over https get a `Strict-Transport-Security` header.
	HSTS bool
}

const (
	DefaultClientIPHeader = "X-Httpbun-Forwarded-For"
	DefaultSchemeHeader   = "X-Httpbun-Forwarded-Proto"
)

var DefaultAllowedRedirectDomains = []string{"example.com", "httpbun.com"}

var listSplitter = regexp.MustCompile(`\s*,\s*|\s+`)

// ParseList parses a comma or whitespace separated list.
func ParseList(raw string) []string {
	items := []string{}
	for _, item := range listSplitter.Split(strings.TrimSpace(raw), -1) {
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func (spec Spec) ClientIPHeaderName() string {
	if spec.ClientIPHeader != "" {
		return spec.ClientIPHeader
	}
	return DefaultClientIPHeader
}

func (spec Spec) SchemeHeaderName() string {
	if spec.SchemeHeader != "" {
		return spec.SchemeHeader
	}
	return DefaultSchemeHeader
}

// IsHiddenHeader tells if a request header should be left out of responses that echo request headers.
func (spec Spec) IsHiddenHeader(name string) bool {
	lowerName := strings.ToLower(name)
	if strings.HasPrefix(lowerName, "x-httpbun-") ||
		strings.EqualFold(name, spec.ClientIPHeaderName()) ||
		strings.EqualFold(name, spec.SchemeHeaderName()) {
		return true
	}
	for _, pattern := range spec.HideHeaders {
		if matched, _ := path.Match(strings.ToLower(pattern), lowerName); matched {
			return true
		}
	}
	return false
}

// IsAnyHost tells if requests to the given host, which may include a port, should behave like `/any`.
func (spec Spec) IsAnyHost(host string) bool {
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		host = hostname
	}
	for _, anyHost := range spec.AnyHosts {
		if strings.EqualFold(host, anyHost) {
			return true
		}
	}
	return false
}

func ParseArgs() Spec {
	spec := &Spec{
		Commit:      Commit,
		CommitShort: util.CommitHashShorten(Commit),
		Date:        Date,
	}

	flag.StringVar(&spec.BindTarget, "bind", os.Getenv("HTTPBUN_BIND"), "Bind target for the server to listen on")
	flag.StringVar(&spec.PathPrefix, "path-prefix", "", "Prefix at which to serve the httpbun APIs")
	flag.BoolVar(&spec.RootIsAny, "root-is-any", false, "Have _all_ endpoints behave like `/any`")
	flag.IntVar(&spec.EndpointBytesSizeLimit, "endpoint-bytes-size-limit", 90, "Size limit on the /bytes endpoint, in number of bytes")
	flag.StringVar(&spec.ClientIPHeader, "client-ip-header", "", "Header with the client's IP, set by a proxy in front (default "+DefaultClientIPHeader+")")
	flag.StringVar(&spec.SchemeHeader, "scheme-header", "", "Header with the client's scheme, set by a proxy in front (default "+DefaultSchemeHeader+")")
	hideHeaders := flag.String("hide-headers", "", "Comma separated request headers to leave out of responses, with `*` wildcards")
	anyHosts := flag.String("any-hosts", "", "Comma separated hosts on which all endpoints behave like `/any`")
	flag.BoolVar(&spec.HSTS, "hsts", false, "Send a Strict-Transport-Security header on HTML pages served over https")
	flag.Parse()

	applyEnv(spec, os.Environ())

	// Flags win over env variables.
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "client-ip-header":
			spec.ClientIPHeader = f.Value.String()
		case "scheme-header":
			spec.SchemeHeader = f.Value.String()
		case "hide-headers":
			spec.HideHeaders = ParseList(*hideHeaders)
		case "any-hosts":
			spec.AnyHosts = ParseList(*anyHosts)
		case "hsts":
			spec.HSTS = f.Value.String() == "true"
		}
	})

	spec.PathPrefix = strings.Trim(spec.PathPrefix, "/")
	if spec.PathPrefix != "" {
		spec.PathPrefix = "/" + spec.PathPrefix
	}

	return *spec
}

// applyEnv sets the configuration that comes from env variables, given as `NAME=value` items, like os.Environ.
func applyEnv(spec *Spec, environ []string) {
	spec.InfoEnv = map[string]string{}

	for _, e := range environ {
		name, value, _ := strings.Cut(e, "=")
		switch {
		case name == "HTTPBUN_TLS_CERT":
			spec.TLSCertFile = value
		case name == "HTTPBUN_TLS_KEY":
			spec.TLSKeyFile = value
		case name == "HTTPBUN_ALLOWED_REDIRECT_DOMAINS":
			spec.AllowedRedirectDomains = ParseList(value)
		case name == "HTTPBUN_CLIENT_IP_HEADER":
			spec.ClientIPHeader = strings.TrimSpace(value)
		case name == "HTTPBUN_SCHEME_HEADER":
			spec.SchemeHeader = strings.TrimSpace(value)
		case name == "HTTPBUN_HIDE_HEADERS":
			spec.HideHeaders = ParseList(value)
		case name == "HTTPBUN_ANY_HOSTS":
			spec.AnyHosts = ParseList(value)
		case name == "HTTPBUN_HSTS":
			spec.HSTS = value == "1" || strings.EqualFold(value, "true")
		case strings.HasPrefix(name, "HTTPBUN_INFO_"):
			spec.InfoEnv[name] = value
		}
	}
}
