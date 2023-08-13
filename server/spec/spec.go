package spec

import (
	"github.com/sharat87/httpbun/util"
	"log"
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

func ParseArgs(args []string) Spec {
	spec := &Spec{
		Commit:      Commit,
		CommitShort: util.CommitHashShorten(Commit),
		Date:        Date,
	}

	bindTarget := os.Getenv("HTTPBUN_BIND")

	i := 0

	for i < len(args) {
		arg := args[i]

		if arg == "--bind" {
			i++
			bindTarget = args[i]

		} else if arg == "--path-prefix" {
			i++
			spec.PathPrefix = args[i]

		} else {
			log.Fatalf("Unknown argument '%v'", arg)

		}

		i++
	}

	if bindTarget == "" {
		bindTarget = ":3090"
	}

	spec.BindTarget = bindTarget

	spec.PathPrefix = strings.Trim(spec.PathPrefix, "/")
	if spec.PathPrefix != "" {
		spec.PathPrefix = "/" + spec.PathPrefix
	}

	return *spec
}
