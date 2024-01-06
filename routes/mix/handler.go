package mix

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/exchange"
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
	dir  string
	args []string
}

var Routes = map[string]exchange.HandlerFn{
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

var templateFuncMap = template.FuncMap{
	"seq": func(args ...int) []int {
		var start, end, delta int
		switch len(args) {
		case 1:
			start = 0
			end = args[0]
			delta = 1
		case 2:
			start = args[0]
			end = args[1]
			delta = 1
		case 3:
			start = args[0]
			end = args[1]
			delta = args[2]
		}
		if (start > end && delta > 0) || (start < end && delta < 0) {
			delta = -delta
		}
		var seq []int
		for i := start; i != end; i += delta {
			seq = append(seq, i)
		}
		return seq
	},
	"toJSON": func(v any) string {
		buffer := &bytes.Buffer{}
		encoder := json.NewEncoder(buffer)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(v)
		if err != nil {
			log.Printf("Error encoding JSON: %v", err)
			return err.Error()
		}
		return string(bytes.TrimSpace(buffer.Bytes()))
	},
}

func computeMixEntries(ex *exchange.Exchange) ([]entry, error) {
	// We need raw path here, with percent encoding intact.
	// TODO: trim also the base path.
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

func handleMix(ex *exchange.Exchange) {
	entries, err := computeMixEntries(ex)
	if err != nil {
		ex.RespondBadRequest(err.Error())
		return
	}

	var status int
	headers := http.Header{}
	var cookies map[string]string
	var deleteCookies []string
	var redirectTo string
	var payload []byte
	var delay time.Duration

	for _, entry := range entries {
		switch entry.dir {

		case "s":
			value := entry.args[0]
			codes := regexp.MustCompile("\\d+").FindAllString(value, -1)

			var code string
			if len(codes) > 1 {
				code = codes[rand.Intn(len(codes))]
			} else {
				code = codes[0]
			}

			status, err = strconv.Atoi(code)
			if err != nil {
				ex.RespondBadRequest(err.Error())
				return
			}

		case "h":
			headerValue, err := url.QueryUnescape(entry.args[1])
			if err != nil {
				ex.RespondBadRequest(err.Error())
				return
			}
			headers.Add(entry.args[0], headerValue)

		case "c":
			cookies[entry.args[0]] = entry.args[1]

		case "cd":
			deleteCookies = append(deleteCookies, entry.args[0])

		case "r":
			redirectTo = entry.args[0]

		case "b64":
			payload, err = base64.StdEncoding.DecodeString(entry.args[0])
			if err != nil {
				ex.RespondBadRequest(err.Error())
				return
			}

		case "d":
			seconds, err := strconv.ParseFloat(entry.args[0], 32)
			if err != nil {
				ex.RespondBadRequest(err.Error())
				return
			}
			if seconds > 10 {
				ex.RespondBadRequest("delay must be less than 10 seconds")
				return
			}
			delay = time.Duration(int(seconds * float64(time.Second)))

		case "t":
			templateContent, err := base64.StdEncoding.DecodeString(entry.args[0])
			payload, err = renderTemplate(ex, string(templateContent))
			if err != nil {
				ex.RespondBadRequest(err.Error())
				return
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

	for key, value := range headers {
		ex.ResponseWriter.Header()[key] = value
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

	ex.ResponseWriter.WriteHeader(status)
	ex.WriteBytes(payload)
}

func handleMixer(ex *exchange.Exchange) {
	entries, err := computeMixEntries(ex)
	if err != nil {
		ex.RespondBadRequest(err.Error())
		return
	}

	assets.Render("mixer.html", *ex, map[string]any{
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
