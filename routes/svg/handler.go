package svg

import (
	"html"
	"net/http"
	"strings"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/util"
)

var Routes = map[string]ex.HandlerFn{
	"/svg/(?P<seed>.+)": handleSVGSeeded,
}

var RouteList = []ex.Route{
	ex.NewRoute(`/svg/(?P<seed>.+)`, handleSVGSeeded),
}

func handleSVGSeeded(ex *ex.Exchange) response.Response {
	seed := ex.Field("seed")

	color := "#" + util.Md5sum(seed)[:6]

	initials := []rune(seed)
	if len(initials) > 2 {
		initials = initials[:2]
	}
	text := html.EscapeString(strings.ToUpper(string(initials)))

	body := `<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
		<circle cx="50%" cy="50%" r="45%" fill="` + color + `" stroke="none" />
		<text x="50%" y="53%" text-anchor="middle" dominant-baseline="middle" font-size="36" font-family="sans-serif" fill="` + util.ComputeFgForBg(color) + `">` + text + `</text>
	</svg>`

	return response.Response{
		Header: http.Header{
			"Content-Type": []string{"image/svg+xml"},
		},
		Body: body,
	}
}
