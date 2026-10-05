package spec

import (
	"flag"
	"os"
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
}

var DefaultAllowedRedirectDomains = []string{"example.com", "httpbun.com"}

var allowedRedirectDomainsSplitter = regexp.MustCompile(`\s*,\s*|\s+`)

// ParseAllowedRedirectDomains parses a comma or whitespace separated list of domains.
func ParseAllowedRedirectDomains(raw string) []string {
	domains := []string{}
	for _, domain := range allowedRedirectDomainsSplitter.Split(strings.TrimSpace(raw), -1) {
		if domain != "" {
			domains = append(domains, domain)
		}
	}
	return domains
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
	flag.Parse()

	applyEnv(spec, os.Environ())

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
			spec.AllowedRedirectDomains = ParseAllowedRedirectDomains(value)
		case strings.HasPrefix(name, "HTTPBUN_INFO_"):
			spec.InfoEnv[name] = value
		}
	}
}
