package mix

import (
	"encoding/base64"
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/mux"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type entry struct {
	dir  string
	args []string
}

var Routes = map[string]mux.HandlerFn{
	`/mix\b(?P<conf>.*)`:   handleMix,
	`/mixer\b(?P<conf>.*)`: handleMixer,
}

func computeMixEntries(ex *exchange.Exchange) ([]entry, error) {
	path := ex.Field("conf")
	query := ex.URL.RawQuery

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
			return entries, err
		}

		switch directive {

		case "s":
			entries = append(entries, entry{"s", []string{value}})

		case "h":
			headerKey, headerValue, _ := strings.Cut(value, ":")
			entries = append(entries, entry{"h", []string{headerKey, headerValue}})

		case "c":
			cookieName, cookieValue, _ := strings.Cut(value, ":")
			entries = append(entries, entry{"c", []string{cookieName, cookieValue}})

		case "cd":
			entries = append(entries, entry{"cd", []string{value}})

		case "r":
			entries = append(entries, entry{"r", []string{value}})

		case "b64":
			entries = append(entries, entry{"b64", []string{value}})

		case "d":
			entries = append(entries, entry{"d", []string{value}})

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
	var headers http.Header
	var cookies map[string]string
	var deleteCookies []string
	var redirectTo string
	var payload []byte
	var delay time.Duration

	for _, entry := range entries {
		switch entry.dir {

		case "s":
			status, err = strconv.Atoi(entry.args[0])
			if err != nil {
				ex.RespondBadRequest(err.Error())
				return
			}

		case "h":
			headers.Add(entry.args[0], entry.args[1])

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
