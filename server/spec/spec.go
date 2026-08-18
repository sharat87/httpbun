package spec

import (
	"flag"
	"os"
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

	spec.PathPrefix = strings.Trim(spec.PathPrefix, "/")
	if spec.PathPrefix != "" {
		spec.PathPrefix = "/" + spec.PathPrefix
	}

	return *spec
}
