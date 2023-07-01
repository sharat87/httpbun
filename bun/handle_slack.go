package bun

import (
	"bytes"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/util"
	"io/ioutil"
	"net/http"
)

func handleSlack(ex *exchange.Exchange) {
	message := "*From*: `" + ex.Request.RemoteAddr + "`\n\n*" + ex.Request.Method + "* `" + ex.Request.URL.String() + "`\n"

	for k, v := range ex.Request.Header {
		message += "*" + k + "*: `" + v[0] + "`\n"
	}
	message += "\n"

	incomingBody, err := ioutil.ReadAll(ex.Request.Body)
	if err == nil {
		if len(incomingBody) > 0 {
			message += "```\n" + string(incomingBody) + "\n```"
		} else {
			message += "_No body._"
		}
	} else {
		message += "_*Error reading body: " + err.Error() + "*_"
	}

	resp, err := http.DefaultClient.Post("https://"+ex.Field("hook"), "application/json", bytes.NewReader(util.ToJsonMust(map[string]any{
		"text": message,
	})))
	if err != nil {
		ex.WriteLn("Error sending message to Slack: " + err.Error())
		return
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ex.WriteLn("Error reading response from Slack: " + err.Error())
		return
	}

	ex.WriteBytes(responseBody)
}
