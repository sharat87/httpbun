package mix

import (
	"bytes"
	"encoding/base64"
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/util"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type entry struct {
	Dir  string   `json:"dir"`
	Args []string `json:"args"`
}

const PatMix = `/mix\b.*`

var Routes = map[string]exchange.HandlerFn{
	PatMix:           handleMix,
	`/mixer\b(/.*)?`: handleMixer,
	`/help/mixer`:    handleMixerHelp,
}

var singleValueDirectives = map[string]any{
	"s":     nil,
	"cd":    nil,
	"r":     nil,
	"b64":   nil,
	"d":     nil,
	"t":     nil,
	"slack": nil,
}

var pairValueDirectives = map[string]any{
	"h": nil,
	"c": nil,
}

func computeMixEntries(ex *exchange.Exchange) ([]entry, error) {
	// We need raw path here, with percent encoding intact.
	path := strings.TrimPrefix(ex.RoutedPath, "/mix")
	var source, itemSep string
	var unescape func(string) (string, error)

	if path != "" {
		source = path
		itemSep = "/"
		unescape = url.PathUnescape
	} else {
		source = ex.Request.URL.RawQuery
		itemSep = "&"
		unescape = url.QueryUnescape
	}

	var entries []entry

	for _, part := range strings.Split(source, itemSep) {
		if part == "" {
			continue
		}

		if part == "end" {
			break
		}

		directive, value, _ := strings.Cut(part, "=")
		value, err := unescape(value)
		if err != nil {
			log.Printf("Error unescaping %s: %v", value, err)
			return entries, err
		}

		if _, ok := singleValueDirectives[directive]; ok {
			entries = append(entries, entry{directive, []string{value}})

		} else if _, ok := pairValueDirectives[directive]; ok {
			itemName, itemValue, _ := strings.Cut(value, ":")
			entries = append(entries, entry{directive, []string{itemName, itemValue}})

		}

	}

	return entries, nil
}

func handleMix(ex *exchange.Exchange) response.Response {
	entries, err := computeMixEntries(ex)
	if err != nil {
		return response.BadRequest(err.Error())
	}

	res := &response.Response{
		Header: http.Header{},
	}
	var redirectTo string
	var payload []byte
	var delay time.Duration

	for _, entry := range entries {
		switch entry.Dir {

		case "s":
			value := entry.Args[0]
			codes := regexp.MustCompile("\\d+").FindAllString(value, -1)

			var code string
			if len(codes) > 1 {
				code = codes[rand.Intn(len(codes))]
			} else {
				code = codes[0]
			}

			res.Status, err = strconv.Atoi(code)
			if err != nil {
				return response.BadRequest(err.Error())
			}

		case "h":
			headerValue, err := url.QueryUnescape(entry.Args[1])
			if err != nil {
				return response.BadRequest(err.Error())
			}
			res.Header.Add(entry.Args[0], headerValue)

		case "c":
			cookieValue, err := url.QueryUnescape(entry.Args[1])
			if err != nil {
				return response.BadRequest(err.Error())
			}
			res.Cookies = append(res.Cookies, http.Cookie{
				Name:  entry.Args[0],
				Value: cookieValue,
				Path:  "/",
			})

		case "cd":
			res.Cookies = append(res.Cookies, http.Cookie{
				Name:    entry.Args[0],
				Value:   "",
				Path:    "/",
				Expires: time.Unix(0, 0),
				MaxAge:  -1, // This will produce `Max-Age: 0` in the cookie.
			})

		case "r":
			if redirectTo != "" {
				return response.BadRequest("multiple redirects not allowed")
			}
			redirectTo, err = url.QueryUnescape(entry.Args[0])
			if err != nil {
				return response.BadRequest(err.Error())
			}

		case "b64":
			payload, err = base64.StdEncoding.DecodeString(entry.Args[0])
			if err != nil {
				return response.BadRequest(err.Error())
			}

		case "d":
			seconds, err := strconv.ParseFloat(entry.Args[0], 32)
			if err != nil {
				return response.BadRequest(err.Error())
			}
			if seconds > 10 {
				return response.BadRequest("delay must be less than 10 seconds")
			}
			delay = time.Duration(int(seconds * float64(time.Second)))

		case "t":
			templateContent, err := base64.StdEncoding.DecodeString(entry.Args[0])
			payload, err = renderTemplate(ex, string(templateContent))
			if err != nil {
				return response.BadRequest(err.Error())
			}

		case "slack":
			sendRequestToSlack(entry.Args[0], ex)

		}
	}

	if redirectTo != "" {
		if res.Status == 0 {
			res.Status = http.StatusTemporaryRedirect
		}
		res.Header.Set("Location", redirectTo)
	}

	if delay > 0 {
		time.Sleep(delay)
	}

	if _, ok := res.Header["Content-Length"]; !ok {
		res.Header.Set("Content-Length", strconv.Itoa(len(payload)))
	}

	if len(payload) > 0 {
		res.Body = payload
	}

	return *res
}

func handleMixer(ex *exchange.Exchange) response.Response {
	return assets.Render("mixer.html", *ex, nil)
}

func handleMixerHelp(ex *exchange.Exchange) response.Response {
	return assets.Render("mixer-help.html", *ex, nil)
}

func renderTemplate(ex *exchange.Exchange, templateContent string) ([]byte, error) {
	tpl, err := template.New("mix").Funcs(templateFuncMap).Parse(templateContent)
	if err != nil {
		ex.Finish(response.BadRequest(err.Error()))
		return nil, err
	}
	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, nil)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sendRequestToSlack(param string, ex *exchange.Exchange) {
	message := "*From*: `" + ex.Request.RemoteAddr + "`\n\n```\n" + ex.Request.Method + " " + ex.FullUrl() + "\n"

	for k, v := range ex.Request.Header {
		message += k + ": " + v[0] + "\n"
	}

	incomingBody, err := io.ReadAll(ex.Request.Body)
	if err == nil {
		if len(incomingBody) > 0 {
			message += "\n" + string(incomingBody) + "\n```\n"
		} else {
			message += "```\n\n_No request body._\n"
		}
	} else {
		message += "```\n\n_*Error reading body: " + err.Error() + "*_\n"
	}

	message += "\n-- Httpbun (<" + ex.FindScheme() + "://" + ex.Request.Host + "|" + ex.Request.Host + ">)\n"

	if strings.HasPrefix(param, "xoxb-") {
		// param is Slack API token
		// not supported yet
	} else if strings.Count(param, "/") == 2 {
		// param is Slack webhook URL
		resp, err := http.DefaultClient.Post(
			"https://hooks.slack.com/services/"+param,
			c.ApplicationJSON,
			bytes.NewReader(util.ToJsonMust(map[string]any{
				"text": message,
			})),
		)
		if err != nil {
			log.Printf("Error sending message to Slack: %v :: %v", err, resp)
		}
	}
}
