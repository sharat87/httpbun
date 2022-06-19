package bun

import (
	"fmt"
	"github.com/sharat87/httpbun/exchange"
	"net/http"
	"regexp"
	"strconv"
)

func handleRedirectTo(ex *exchange.Exchange) {
	urls := ex.Request.URL.Query()["url"]
	if len(urls) < 1 || urls[0] == "" {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ex.WriteLn("Need url parameter")
		return
	}

	statusCodes := ex.Request.URL.Query()["status_code"]
	statusCode := http.StatusFound
	if statusCodes != nil {
		var err error
		if statusCode, err = strconv.Atoi(statusCodes[0]); err != nil {
			ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
			ex.WriteLn("status_code must be an integer")
			return
		}
		if statusCode < 300 || statusCode > 399 {
			statusCode = 302
		}
	}

	ex.ResponseWriter.Header().Set("Location", urls[0])
	ex.ResponseWriter.WriteHeader(statusCode)
}

func handleAbsoluteRedirect(ex *exchange.Exchange) {
	n, _ := strconv.Atoi(ex.Field("count"))

	if n > MaxRedirectCount {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ex.WriteF("No more than %v redirects allowed.\n", MaxRedirectCount)
	} else if n > 1 {
		ex.Redirect(ex.ResponseWriter, regexp.MustCompile("/\\d+$").ReplaceAllLiteralString(ex.Request.URL.String(), "/"+fmt.Sprint(n-1)), false)
	} else {
		ex.Redirect(ex.ResponseWriter, "/get", false)
	}
}

func handleRelativeRedirect(ex *exchange.Exchange) {
	n, _ := strconv.Atoi(ex.Field("count"))

	if n > MaxRedirectCount {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ex.WriteF("No more than %v redirects allowed.\n", MaxRedirectCount)
	} else if n > 1 {
		ex.Redirect(ex.ResponseWriter, regexp.MustCompile("/\\d+$").ReplaceAllLiteralString(ex.URL.Path, "/"+fmt.Sprint(n-1)), true)
	} else {
		ex.Redirect(ex.ResponseWriter, "/get", true)
	}
}
