package spec

import (
	"flag"
	"github.com/sharat87/httpbun/util"
	"os"
	"strings"
)

var (
	Commit string
	Date   string
)

type Spec struct {
	BindTarget  string
	PathPrefix  string
	Commit      string
	CommitShort string
	Date        string
}

func ParseArgs() Spec {
	spec := &Spec{
		Commit:      Commit,
		CommitShort: util.CommitHashShorten(Commit),
		Date:        Date,
	}

	flag.StringVar(&spec.BindTarget, "bind", os.Getenv("HTTPBUN_BIND"), "Bind target for the server to listen on")
	flag.StringVar(&spec.PathPrefix, "path-prefix", "", "Prefix at which to serve the httpbun APIs")
	flag.Parse()

	if spec.BindTarget == "" {
		spec.BindTarget = ":3090"
	}

	spec.PathPrefix = strings.Trim(spec.PathPrefix, "/")
	if spec.PathPrefix != "" {
		spec.PathPrefix = "/" + spec.PathPrefix
	}

	return *spec
}
