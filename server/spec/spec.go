package spec

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"regexp"
	"strconv"
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

	// Most requests handled at the same time. More get a 503 right away. When zero, DefaultMaxConcurrentRequests is
	// used.
	MaxConcurrentRequests int

	// Largest request body allowed, in bytes. Bigger bodies get a 413. When zero, DefaultMaxBodySize is used.
	MaxBodySize int64
}

const (
	DefaultClientIPHeader = "X-Httpbun-Forwarded-For"
	DefaultSchemeHeader   = "X-Httpbun-Forwarded-Proto"

	// Httpbun isn't meant for high load, so these limits are always on, and keep any single client or burst from
	// exhausting the process.
	DefaultMaxConcurrentRequests       = 256
	DefaultMaxBodySize           int64 = 10 << 20
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

func (spec Spec) MaxConcurrentRequestsLimit() int {
	if spec.MaxConcurrentRequests > 0 {
		return spec.MaxConcurrentRequests
	}
	return DefaultMaxConcurrentRequests
}

func (spec Spec) MaxBodySizeLimit() int64 {
	if spec.MaxBodySize > 0 {
		return spec.MaxBodySize
	}
	return DefaultMaxBodySize
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
	maxConcurrentRequests := flag.Int("max-concurrent-requests", DefaultMaxConcurrentRequests, "Most requests handled at the same time, more get a 503")
	maxBodySize := flag.Int64("max-body-size", DefaultMaxBodySize, "Largest request body allowed, in bytes, bigger ones get a 413")
	flag.Parse()

	if err := applyEnv(spec, os.Environ()); err != nil {
		log.Fatal(err)
	}

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
		case "max-concurrent-requests":
			if *maxConcurrentRequests <= 0 {
				log.Fatalf("--max-concurrent-requests must be a positive integer, got %d", *maxConcurrentRequests)
			}
			spec.MaxConcurrentRequests = *maxConcurrentRequests
		case "max-body-size":
			if *maxBodySize <= 0 {
				log.Fatalf("--max-body-size must be a positive integer, got %d", *maxBodySize)
			}
			spec.MaxBodySize = *maxBodySize
		}
	})

	spec.PathPrefix = strings.Trim(spec.PathPrefix, "/")
	if spec.PathPrefix != "" {
		spec.PathPrefix = "/" + spec.PathPrefix
	}

	return *spec
}

// applyEnv sets the configuration that comes from env variables, given as `NAME=value` items, like os.Environ.
func applyEnv(spec *Spec, environ []string) error {
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
		case name == "HTTPBUN_MAX_CONCURRENT_REQUESTS":
			limit, err := parsePositiveInt(name, value)
			if err != nil {
				return err
			}
			spec.MaxConcurrentRequests = int(limit)
		case name == "HTTPBUN_MAX_BODY_SIZE":
			limit, err := parsePositiveInt(name, value)
			if err != nil {
				return err
			}
			spec.MaxBodySize = limit
		case strings.HasPrefix(name, "HTTPBUN_INFO_"):
			spec.InfoEnv[name] = value
		}
	}
	return nil
}

func parsePositiveInt(name, raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer, got %q", name, raw)
	}
	return value, nil
}
