package bun

import (
	"github.com/sharat87/httpbun/exchange"
	"strings"
)

func handleFrame(ex *exchange.Exchange) {
	embedUrl, _ := ex.QueryParamSingle("url")

	ex.ResponseWriter.Header().Set("Content-Type", "text/html")

	warning := ""
	if ex.URL.Scheme == "http" && strings.HasPrefix(embedUrl, "https://") {
		warning = `
		<p>You are embedding an https URL inside an http page, switch to full https for best experience.
		<a href='#' onclick='location.protocol = "https:"'>Click here to switch</a>.</p>`
	}

	ex.WriteF(`<!doctype html>
<html>
<style>
* { box-sizing: border-box }
html, body, form { margin: 0; min-height: 100vh }
form { display: flex; flex-direction: column }
iframe { border: none; flex-grow: 1 }
input { font-size: 1.2em; width: calc(100%% - 1em); margin: .5em }
p { margin: .5em }
</style>
<form>
<input name=url value='%s' placeholder='Enter URL to embed in an iframe' autofocus required>
%s
<button style='display:none'>Embed</button>
<iframe src="%s"></iframe>
</form>
`, embedUrl, warning, embedUrl)
}
