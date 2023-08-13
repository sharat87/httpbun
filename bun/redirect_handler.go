package bun

import (
	"fmt"
	"github.com/sharat87/httpbun/exchange"
	"net/http"
	"strconv"
)

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
	mode := ex.Field("mode")
	n, _ := strconv.Atoi(ex.Field("count"))

	if n > MaxRedirectCount {
		ex.RespondBadRequest("No more than %v redirects allowed.", MaxRedirectCount)

	} else if n > 1 {
		target := fmt.Sprint(n - 1)
		if mode == "absolute-" {
			target = "/absolute-redirect/" + target
		}
		ex.Redirect(ex.ResponseWriter, target)

	} else {
		ex.Redirect(ex.ResponseWriter, "/anything")

	}
}
