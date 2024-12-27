package spec

import (
	"flag"
	"fmt"
	"os"
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

	// A banner to show on the homepage.
	Banner   string
	BannerBg string
	BannerFg string

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
	flag.BoolVar(&spec.RootIsAny, "root-is-any", false, "Have _all_ endpoints behave like `/any`")
	flag.StringVar(&spec.Banner, "banner", "", "A banner text to display on the homepage")
	flag.Parse()

	if spec.Banner != "" {
		// A silly way to reproducibly turn a piece of text into a color.
		color := "#" + util.Md5sum(spec.Banner)[:6]
		spec.BannerBg = color
		if isLight(color) {
			spec.BannerFg = "#222"
		} else {
			spec.BannerFg = "#eeee"
		}
	}

	spec.PathPrefix = strings.Trim(spec.PathPrefix, "/")
	if spec.PathPrefix != "" {
		spec.PathPrefix = "/" + spec.PathPrefix
	}

	return *spec
}

func isLight(c string) bool {
	rgb, err := strconv.ParseInt(c[1:], 16, 32)
	if err != nil {
		fmt.Println("Error parsing color:", err)
		return false
	}
	r := (rgb >> 16) & 0xff
	g := (rgb >> 8) & 0xff
	b := rgb & 0xff

	luma := 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b) // per ITU-R BT.709

	return luma < 40
}
