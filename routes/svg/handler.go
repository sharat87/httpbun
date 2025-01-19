package svg

import (
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/util"
	"net/http"
	"strings"
)

var Routes = map[string]exchange.HandlerFn{
	"/svg/(?P<seed>.+)": handleSVGSeeded,
}

func handleSVGSeeded(ex *exchange.Exchange) response.Response {
	seed := ex.Field("seed")

	color := "#" + util.Md5sum(seed)[:6]

	body := `<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
		<circle cx="50%" cy="50%" r="45%" fill="` + color + `" stroke="none" />
		<text x="50%" y="53%" text-anchor="middle" dominant-baseline="middle" font-size="36" font-family="sans-serif" fill="` + util.ComputeFgForBg(color) + `">` + strings.ToUpper(seed[:2]) + `</text>
	</svg>`

	return response.Response{
		Header: http.Header{
			"Content-Type": []string{"image/svg+xml"},
		},
		Body: body,
	}
}
