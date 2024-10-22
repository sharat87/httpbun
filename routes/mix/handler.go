package mix

import (
	"bytes"
	"encoding/base64"
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
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

var Routes = map[string]exchange.HandlerFn2{
	`/mix\b(?P<conf>.*)`:   handleMix,
	`/mixer\b(?P<conf>.*)`: handleMixer,
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

func computeMixEntries(ex *exchange.Exchange) ([]entry, error) {
	// We need raw path here, with percent encoding intact.
	path := strings.TrimPrefix(ex.RoutedRawPath, "/mix")
	query := ex.Request.URL.RawQuery

	var source, itemSep string
	var unescape func(string) (string, error)

	if path != "" {
		source = path
		itemSep = "/"
		unescape = url.PathUnescape
	} else {
		source = query
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

	var status int
	headers := http.Header{}
	cookies := map[string]string{}
	var deleteCookies []string
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

			status, err = strconv.Atoi(code)
			if err != nil {
				return response.BadRequest(err.Error())
			}

		case "h":
			headerValue, err := url.QueryUnescape(entry.Args[1])
			if err != nil {
				return response.BadRequest(err.Error())
			}
			headers.Add(entry.Args[0], headerValue)

		case "c":
			cookieValue, err := url.QueryUnescape(entry.Args[1])
			if err != nil {
				return response.BadRequest(err.Error())
			}
			cookies[entry.Args[0]] = cookieValue

		case "cd":
			deleteCookies = append(deleteCookies, entry.Args[0])

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

		}
	}

	if redirectTo != "" {
		if status == 0 {
			status = http.StatusTemporaryRedirect
		}
		headers.Set("Location", redirectTo)
	}

	if status == 0 {
		status = http.StatusOK
	}

	if delay > 0 {
		time.Sleep(delay)
	}

	if _, ok := headers["Content-Length"]; !ok {
		if headers == nil {
			headers = http.Header{}
		}
		headers.Set("Content-Length", strconv.Itoa(len(payload)))
	}

	for key, value := range cookies {
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  key,
			Value: value,
			Path:  "/",
		})
	}

	for _, name := range deleteCookies {
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:    name,
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
			MaxAge:  -1, // This will produce `Max-Age: 0` in the cookie.
		})
	}

	return response.New(status, headers, payload)
}

func handleMixer(ex *exchange.Exchange) response.Response {
	entries, err := computeMixEntries(ex)
	if err != nil {
		return response.BadRequest(err.Error())
	}

	return assets.Render2("mixer.html", *ex, map[string]any{
		"mixEntries": entries,
	})
}

func renderTemplate(ex *exchange.Exchange, templateContent string) ([]byte, error) {
	tpl, err := template.New("mix").Funcs(templateFuncMap).Parse(templateContent)
	if err != nil {
		ex.RespondBadRequest(err.Error())
		return nil, err
	}
	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, nil)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
