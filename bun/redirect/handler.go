package redirect

import (
	"fmt"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/mux"
	"net/http"
	"strconv"
)

const MaxRedirectCount = 20

var Routes = map[string]mux.HandlerFn{
	`/redirect(-to)?/?`:                            handleRedirectTo,
	`/(?P<mode>relative-)?redirect/(?P<count>\d+)`: handleRedirectCount,
	`/(?P<mode>absolute-)redirect/(?P<count>\d+)`:  handleRedirectCount,
}

func handleRedirectTo(ex *exchange.Exchange) {
	query := ex.Request.URL.Query()
	urls := query["url"]
	if len(urls) < 1 || urls[0] == "" {
		ex.RespondBadRequest("Need url parameter")
		return
	}

	statusCodes := query["status_code"]
	if statusCodes == nil {
		statusCodes = query["status"]
	}

	statusCode := http.StatusFound
	if statusCodes != nil {
		var err error
		if statusCode, err = strconv.Atoi(statusCodes[0]); err != nil {
			ex.RespondBadRequest("status_code must be an integer")
			return
		}
		if statusCode < 300 || statusCode > 399 {
			statusCode = 302
		}
	}

	ex.ResponseWriter.Header().Set("Location", urls[0])
	ex.ResponseWriter.WriteHeader(statusCode)
}

func handleRedirectCount(ex *exchange.Exchange) {
	isAbsolute := ex.Field("mode") == "absolute-"
	n, _ := strconv.Atoi(ex.Field("count"))

	if n < 0 {
		ex.RespondBadRequest("count must be a non-negative integer")

	} else if n > MaxRedirectCount {
		ex.RespondBadRequest("count cannot be greater than %v", MaxRedirectCount)

	} else if n > 1 {
		target := fmt.Sprint(n - 1)
		if isAbsolute {
			target = "/absolute-redirect/" + target
		}
		ex.Redirect(target)

	} else {
		var target string
		if isAbsolute {
			target = "/anything"
		} else {
			target = "../anything"
		}
		ex.Redirect(target)

	}
}
