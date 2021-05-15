package util

import (
	"crypto/md5"
	crypto_rand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sharat87/httpbun/request"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var hiddenHeaders = map[string]bool{
	"Total-Route-Time":  true,
	"Via":               true,
	"X-Forwarded-For":   true,
	"X-Forwarded-Port":  true,
	"X-Forwarded-Proto": true,
	"X-Request-Id":      true,
	"X-Request-Start":   true,
}

func HeaderValue(req request.Request, name string) string {
	if values := req.Header[name]; values != nil && len(values) > 0 {
		return values[len(values)-1]
	}

	return ""
}

func Redirect(w http.ResponseWriter, req *request.Request, path string) {
	if strings.HasPrefix(path, "/") {
		path = strings.Repeat("../", strings.Count(req.URL.Path, "/")-1) + strings.TrimPrefix(path, "/")
	}

	w.Header().Set("Location", path)
	w.WriteHeader(http.StatusFound)

	fmt.Fprintf(w, `<!doctype html>
<title>Redirecting...</title>
<h1>Redirecting...</h1>
<p>You should be redirected automatically to target URL: <a href=%q>/cookies</a>.  If not click the link.</p>`, path)
}

func QueryParamInt(req *request.Request, name string, value int) (int, error) {
	args := req.URL.Query()
	var err error

	if len(args[name]) > 0 {
		value, err = strconv.Atoi(args[name][0])
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", name)
		}
	}

	return value, nil
}

func WriteJson(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, ToJsonMust(data))
}

func ToJsonMust(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}

func Md5sum(text string) string {
	// Source: <https://stackoverflow.com/a/25286918/151048>.
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func RandomBytes(n int) []byte {
	b := make([]byte, n)

	if _, err := crypto_rand.Read(b); err != nil {
		fmt.Println("Error: ", err)
		return []byte{}
	}

	return b[:]
}

func RandomString() string {
	return hex.EncodeToString(RandomBytes(16))
}

func Flush(w http.ResponseWriter) bool {
	f, ok := w.(http.Flusher)
	if ok {
		f.Flush()
	}
	return ok
}

func ExposableHeadersMap(req request.Request) map[string]string {
	headers := make(map[string]string)
	for name, values := range req.Header {
		if !hiddenHeaders[name] {
			headers[name] = strings.Join(values, ",")
		}
	}
	return headers
}

func FullUrl(req request.Request) string {
	if !strings.HasPrefix(req.URL.String(), "/") {
		return req.URL.String()
	}

	scheme := "http"
	if os.Getenv("HTTPBUN_SSL_CERT") != "" || HeaderValue(req, "X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	return scheme + "://" + req.Host + req.URL.String()
}

func ParseHeaderValueCsv(content string) []map[string]string {
	data := []map[string]string{}
	if content == "" {
		return data
	}

	runes := []rune(content)
	length := len(runes)
	state := "key-pre"
	key := []rune{}
	val := []rune{}
	isValueJustStarted := false
	inQuotes := false

	currentMap := make(map[string]string)

	for pos := 0; pos < length; pos++ {
		ch := runes[pos]

		if inQuotes {
			if ch == '"' {
				inQuotes = false
			} else if state == "value" {
				val = append(val, ch)
			}

		} else if ch == '=' {
			state = "value"
			isValueJustStarted = true

		} else if ch == ';' || ch == ',' {
			state = "key-pre"
			currentMap[strings.ToLower(string(key))] = string(val)
			key = []rune{}
			val = []rune{}

			if ch == ',' {
				data = append(data, currentMap)
				currentMap = make(map[string]string)
			}

		} else if state == "key-pre" {
			if ch != ' ' {
				// Whitespace just before a key is ignored.
				state = "key"
				key = append(key, ch)
			}

		} else if state == "key" {
			key = append(key, ch)

		} else if state == "value" {
			if isValueJustStarted && ch == '"' {
				inQuotes = true
			} else {
				val = append(val, ch)
			}
			isValueJustStarted = false

		}

	}

	if len(key) > 0 {
		currentMap[strings.ToLower(string(key))] = string(val)
		data = append(data, currentMap)
	}

	return data
}
