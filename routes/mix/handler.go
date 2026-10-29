package mix

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/util"
)

type entry struct {
	Dir  string   `json:"dir"`
	Args []string `json:"args"`
}

const PatMix = `/mix\b.*`

var RouteList = []ex.Route{
	ex.NewRoute(PatMix, handleMix),
	ex.NewRoute(`/mixer\b(/.*)?`, handleMixer),
	ex.NewRoute(`/help/mixer`, handleMixerHelp),
}

var singleValueDirectives = map[string]any{
	"s":   nil,
	"cd":  nil,
	"r":   nil,
	"b64": nil,
	"d":   nil,
	"t":   nil,
}

var pairValueDirectives = map[string]any{
	"h": nil,
	"c": nil,
}

func computeMixEntries(ex *ex.Exchange) ([]entry, error) {
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
		if directive == "slack" {
			// Unknown directives are ignored, but this one sent the request to Slack, so say it's gone, rather than
			// silently not doing that.
			return entries, errors.New("the slack directive has been removed")
		}

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

func handleMix(ex *ex.Exchange) response.Response {
	entries, err := computeMixEntries(ex)
	if err != nil {
		return response.BadRequest("%s", err.Error())
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
			codes := regexp.MustCompile(`\d+`).FindAllString(value, -1)

			if len(codes) == 0 {
				return response.BadRequest("No status codes given")
			}

			var code string
			if len(codes) > 1 {
				code = codes[rand.Intn(len(codes))]
			} else {
				code = codes[0]
			}

			res.Status, err = util.ParseStatusCode(code)
			if err != nil {
				return response.BadRequest("%s", err.Error())
			}

		case "h":
			res.Header.Add(entry.Args[0], entry.Args[1])

		case "c":
			res.Cookies = append(res.Cookies, http.Cookie{
				Name:  entry.Args[0],
				Value: entry.Args[1],
				Path:  "/",
			})

		case "cd":
			res.Cookies = append(res.Cookies, http.Cookie{
				Name:    entry.Args[0],
				Path:    "/",
				Expires: time.Unix(0, 0),
				MaxAge:  -1, // This will produce `Max-Age: 0` in the cookie.
			})

		case "r":
			if redirectTo != "" {
				return response.BadRequest("multiple redirects not allowed")
			}
			// Already unescaped when parsing the directives.
			redirectTo = entry.Args[0]

		case "b64":
			payload, err = base64.StdEncoding.DecodeString(entry.Args[0])
			if err != nil {
				return response.BadRequest("%s", err.Error())
			}

		case "d":
			seconds, err := strconv.ParseFloat(entry.Args[0], 32)
			if err != nil {
				if strings.HasSuffix(err.Error(), "invalid syntax") {
					return response.BadRequest("invalid delay value: '%s'", entry.Args[0])
				} else {
					return response.BadRequest("%s", err.Error())
				}
			}
			if seconds < 0 {
				return response.BadRequest("delay must be a positive number")
			} else if seconds > 10 {
				return response.BadRequest("delay must be less than 10 seconds")
			}
			delay = time.Duration(int(seconds * float64(time.Second)))

		case "t":
			templateContent, err := base64.StdEncoding.DecodeString(entry.Args[0])
			if err != nil {
				return response.BadRequest("%s", err.Error())
			}
			payload, err = renderTemplate(string(templateContent))
			if err != nil {
				return response.BadRequest("%s", err.Error())
			}

		}
	}

	if redirectTo != "" {
		if res.Status == 0 {
			res.Status = http.StatusTemporaryRedirect
		}
		res.Header.Set("Location", redirectTo)
	}

	if delay > 0 {
		if !ex.HoldLongSlot() {
			return response.LongBusy()
		}
		util.Sleep(ex.Request.Context(), delay)
	}

	if len(payload) > 0 {
		res.Body = payload
		if _, ok := res.Header["Content-Length"]; !ok {
			res.Header.Set("Content-Length", strconv.Itoa(len(payload)))
		}
	}

	return *res
}

func handleMixer(ex *ex.Exchange) response.Response {
	return assets.Render("mixer.html", *ex, nil)
}

func handleMixerHelp(ex *ex.Exchange) response.Response {
	return assets.Render("mixer-help.html", *ex, nil)
}

// The most a template can render, so a template with nested loops can't use up all memory.
const maxTemplateOutput = 1 << 20

func renderTemplate(templateContent string) ([]byte, error) {
	funcs := templateFuncs(&workBudget{})
	tpl, err := template.New("mix").Funcs(funcs).Parse(templateContent)
	if err != nil {
		return nil, err
	}
	if err := chargeForWork(tpl, funcs); err != nil {
		return nil, err
	}
	out := &cappedBuffer{limit: maxTemplateOutput}
	err = tpl.Execute(out, nil)
	if errors.Is(err, errTooMuchWork) {
		// Without where in the template it happened, which is often in the charges added by chargeForWork.
		return nil, errTooMuchWork
	} else if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// cappedBuffer is a bytes.Buffer that errors on writes that would take it past its limit, which stops a template.
type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("template output can't be more than %d bytes", b.limit)
	}
	return b.Buffer.Write(p)
}
