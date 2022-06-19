package bun

import (
	"fmt"
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/exchange"
	"net/http"
	"net/url"
	"strings"
)

func handleOauthAuthorize(ex *exchange.Exchange) {
	// Ref: <https://datatracker.ietf.org/doc/html/rfc6749>.
	fmt.Println("oauth auth")

	// TODO: Handle POST also, where params are read from the body.
	if ex.Request.Method != http.MethodGet {
		ex.RespondError(http.StatusMethodNotAllowed)
		return
	}

	var errors []string
	params := ex.Request.URL.Query()

	redirectUrl, err := ex.QueryParamSingle("redirect_uri")
	if err != nil {
		errors = append(errors, err.Error())
	} else if !strings.HasPrefix(redirectUrl, "http://") && !strings.HasPrefix(redirectUrl, "https://") {
		errors = append(errors, "The `redirect_uri` must be an absolute URL, and should start with `http://` or `https://`.")
	}

	responseType, err := ex.QueryParamSingle("response_type")
	if err != nil {
		errors = append(errors, err.Error())
	}

	// clientId, err := ex.QueryParamSingle("client_id")
	// if err != nil {
	// 	// Required if responseType is "code" or "token"
	// 	errors = append(errors, err.Error())
	// }

	state := ""
	if len(params["state"]) > 0 {
		state = params["state"][0]
	}

	var scopes []string
	if len(params["scope"]) > 0 {
		scopes = strings.Split(strings.Join(params["scope"], " "), " ")
	}

	if len(errors) > 0 {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
	}

	// TODO: Error handling as per <https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2.1>.
	assets.Render("oauth-consent.html", ex.ResponseWriter, map[string]interface{}{
		"Errors":       errors,
		"scopes":       scopes,
		"redirectUrl":  redirectUrl,
		"responseType": responseType,
		"state":        state,
	})
}

func handleOauthAuthorizeSubmit(ex *exchange.Exchange) {
	if ex.Request.Method != http.MethodPost {
		ex.RespondError(http.StatusMethodNotAllowed)
		return
	}

	// TODO: Error out if there's *any* query params here.
	err := ex.Request.ParseForm()
	if err != nil {
		ex.RespondError(http.StatusBadRequest)
		return
	}

	decision, _ := ex.FormParamSingle("decision")

	redirectUrl, _ := ex.FormParamSingle("redirect_uri")
	responseType, _ := ex.FormParamSingle("response_type")
	state, _ := ex.FormParamSingle("state")

	var params []string

	if state != "" {
		params = append(params, "state="+url.QueryEscape(state))
	}

	if len(ex.Request.Form["scope"]) > 0 {
		params = append(params, "scope="+url.QueryEscape(strings.Join(ex.Request.Form["scope"], " ")))
	}

	if decision == "Approve" {
		if responseType == "code" {
			params = append(params, "code=123")
		} else if responseType == "token" {
			params = append(params, "access_token=456")
			params = append(params, "token_type=bearer")
		} else {
			params = append(params, "approved=true")
		}
	} else {
		params = append(params, "error=access_denied")
	}

	ex.Redirect(ex.ResponseWriter, redirectUrl+"?"+strings.Join(params, "&"), true)
}
