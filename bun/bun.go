package bun

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/mux"
	"github.com/sharat87/httpbun/util"
	"io/ioutil"
	"log"
	"math/rand"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxRedirectCount = 20

func MakeBunHandler(pathPrefix, commit, date string) mux.Mux {
	m := mux.Mux{
		PathPrefix: pathPrefix,
	}

	m.HandleFunc(`/(index\.html)?`, func(ex *exchange.Exchange) {
		ex.ResponseWriter.Header().Set("Content-Type", "text/html")
		assets.Render("index.html", ex.ResponseWriter, map[string]string{
			"host":        ex.URL.Host,
			"commit":      commit,
			"commitShort": util.CommitHashShorten(commit),
			"date":        date,
		})
	})

	m.HandleFunc(`/(?P<name>.+\.(png|ico|webmanifest))`, func(ex *exchange.Exchange) {
		assets.WriteAsset(ex.Field("name"), ex.ResponseWriter, ex.Request)
	})

	m.HandleFunc("/health", handleHealth)

	m.HandleFunc("/get", handleValidMethod)
	m.HandleFunc("/post", handleValidMethod)
	m.HandleFunc("/put", handleValidMethod)
	m.HandleFunc("/patch", handleValidMethod)
	m.HandleFunc("/delete", handleValidMethod)

	m.HandleFunc("/headers", handleHeaders)

	m.HandleFunc("/payload", handlePayload)

	m.HandleFunc("/basic-auth/(?P<user>[^/]+)/(?P<pass>[^/]+)/?", handleAuthBasic)
	m.HandleFunc("/bearer(/(?P<tok>\\w+))?", handleAuthBearer)
	m.HandleFunc("/digest-auth/(?P<qop>[^/]+)/(?P<user>[^/]+)/(?P<pass>[^/]+)/?", handleAuthDigest)
	m.HandleFunc("/digest-auth/(?P<user>[^/]+)/(?P<pass>[^/]+)/?", handleAuthDigest)

	m.HandleFunc("/status/(?P<codes>[\\d,]+)", handleStatus)
	m.HandleFunc("/ip(\\.(?P<format>txt|json))?", handleIp)
	m.HandleFunc("/user-agent", handleUserAgent)

	m.HandleFunc("/cache", handleCache)
	m.HandleFunc("/cache/(?P<age>\\d+)", handleCacheControl)
	m.HandleFunc("/etag/(?P<etag>[^/]+)", handleEtag)
	m.HandleFunc("/(response|respond-with)-headers?/?", handleResponseHeaders)

	m.HandleFunc("/deny", handleSampleRobotsDeny)
	m.HandleFunc("/html", handleSampleHtml)
	m.HandleFunc("/json", handleSampleJson)
	m.HandleFunc("/robots.txt", handleSampleRobotsTxt)
	m.HandleFunc("/xml", handleSampleXml)
	m.HandleFunc("/image/svg", handleImageSvg)

	m.HandleFunc("/base64(/(?P<encoded>.*))?", handleDecodeBase64)
	m.HandleFunc("/bytes/(?P<size>\\d+)", handleRandomBytes)
	m.HandleFunc("/delay/(?P<delay>[^/]+)", handleDelayedResponse)
	m.HandleFunc("/drip(-(?P<mode>lines))?", handleDrip)
	m.HandleFunc("/links/(?P<count>\\d+)(/(?P<offset>\\d+))?/?", handleLinks)
	m.HandleFunc("/range/(?P<count>\\d+)/?", handleRange)

	m.HandleFunc("/cookies", handleCookies)
	m.HandleFunc("/cookies/delete", handleCookiesDelete)
	m.HandleFunc("/cookies/set(/(?P<name>[^/]+)/(?P<value>[^/]+))?", handleCookiesSet)

	m.HandleFunc("/redirect(-to)?/?", handleRedirectTo)
	m.HandleFunc("/(relative-)?redirect/(?P<count>\\d+)", handleRelativeRedirect)
	m.HandleFunc("/absolute-redirect/(?P<count>\\d+)", handleAbsoluteRedirect)

	m.HandleFunc("/anything\\b.*", handleAnything)

	m.HandleFunc("/mix\\b(?P<conf>.*)", handleMix)

	if os.Getenv("HTTPBUN_INFO_ENABLED") == "1" {
		m.HandleFunc("/info", handleInfo)
	}

	return m
}

func handleHealth(ex *exchange.Exchange) {
	ex.WriteLn("ok")
}

type InfoJsonOptions struct {
	BodyInfo bool
}

func handleValidMethod(ex *exchange.Exchange) {
	allowedMethod := strings.TrimPrefix(ex.URL.Path, "/")
	if !strings.EqualFold(ex.Request.Method, allowedMethod) {
		allowedMethods := strings.ToUpper(allowedMethod) + ", OPTIONS"
		ex.ResponseWriter.Header().Set("Allow", allowedMethods)
		ex.ResponseWriter.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		if ex.Request.Method != http.MethodOptions {
			ex.ResponseWriter.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	isNonGet := ex.Request.Method != http.MethodGet
	sendInfoJson(ex, InfoJsonOptions{
		BodyInfo: isNonGet,
	})
}

func handleAnything(ex *exchange.Exchange) {
	sendInfoJson(ex, InfoJsonOptions{
		BodyInfo: true,
	})
}

func handleHeaders(ex *exchange.Exchange) {
	util.WriteJson(ex.ResponseWriter, map[string]interface{}{
		"headers": ex.ExposableHeadersMap(),
	})
}

func handlePayload(ex *exchange.Exchange) {
	if contentTypeValues, ok := ex.Request.Header["Content-Type"]; ok {
		ex.ResponseWriter.Header().Set("Content-Type", contentTypeValues[0])
	}

	bodyBytes, err := ioutil.ReadAll(ex.CappedBody)
	if err != nil {
		fmt.Println("Error reading request payload", err)
	}

	ex.WriteBytes(bodyBytes)
}

func sendInfoJson(ex *exchange.Exchange, options InfoJsonOptions) {
	args := make(map[string]interface{})
	for name, values := range ex.Request.URL.Query() {
		if len(values) > 1 {
			args[name] = values
		} else {
			args[name] = values[0]
		}
	}

	result := map[string]interface{}{
		"method":  ex.Request.Method,
		"args":    args,
		"headers": ex.ExposableHeadersMap(),
		"origin":  ex.FindIncomingIPAddress(),
		"url":     ex.FullUrl(),
	}

	contentTypeHeaderValue := ex.HeaderValueLast("Content-Type")
	if contentTypeHeaderValue == "" {
		contentTypeHeaderValue = "text/plain"
	}
	contentType, params, err := mime.ParseMediaType(contentTypeHeaderValue)
	if err != nil {
		log.Printf("Error parsing content type %q %v.", ex.HeaderValueLast("Content-Type"), err)
		return
	}

	if options.BodyInfo {
		form := make(map[string]any)
		var jsonData *any
		files := make(map[string]any)
		var data any // string or []byte

		if contentType == "application/x-www-form-urlencoded" {
			body := ex.BodyString()
			if parsed, err := url.ParseQuery(body); err != nil {
				data = body
			} else {
				for name, values := range parsed {
					if len(values) > 1 {
						form[name] = values
					} else {
						form[name] = values[0]
					}
				}
			}

		} else if contentType == "application/json" {
			body := ex.BodyString()
			var result interface{}
			if json.Unmarshal([]byte(body), &result) == nil {
				jsonData = &result
			}
			data = body

		} else if contentType == "multipart/form-data" {
			// This might work for `multipart/mixed` as well. Confirm.
			reader := multipart.NewReader(ex.Request.Body, params["boundary"])
			allFileData, err := reader.ReadForm(32 << 20)
			if err != nil {
				errorMessage := "Error reading multipart form data: " + err.Error()
				ex.RespondError(http.StatusBadRequest, "multipart-read-error", errorMessage)
				log.Println(errorMessage)
				return
			}

			for name, fileHeaders := range allFileData.File {
				fileHeader := fileHeaders[0]
				var content any
				if f, err := fileHeader.Open(); err != nil {
					fmt.Println("Error opening fileHeader", err)
				} else if content, err = ioutil.ReadAll(f); err != nil {
					fmt.Println("Error reading from fileHeader", err)
				} else {
					if utf8.Valid(content.([]byte)) {
						content = string(content.([]byte))
					}
					headers := map[string]string{}
					for name, values := range fileHeader.Header {
						headers[name] = strings.Join(values, ",")
					}
					files[name] = map[string]any{
						"filename": fileHeader.Filename,
						"size":     fileHeader.Size,
						"headers":  headers,
						"content":  content,
					}
				}
			}

			for name, valueInfo := range allFileData.Value {
				form[name] = valueInfo[0]
			}

		} else {
			data = ex.BodyBytes()
			if utf8.Valid(data.([]byte)) {
				data = string(data.([]byte))
			}

		}

		if data == nil {
			data = ""
		}

		result["form"] = form
		result["data"] = data
		result["json"] = jsonData
		result["files"] = files
	}

	util.WriteJson(ex.ResponseWriter, result)
}

func handleStatus(ex *exchange.Exchange) {
	codes := regexp.MustCompile("\\d+").FindAllString(ex.Request.URL.String(), -1)

	var code string
	if len(codes) > 1 {
		code = codes[rand.Intn(len(codes))]
	} else {
		code = codes[0]
	}

	codeNum, _ := strconv.Atoi(code)
	ex.ResponseWriter.WriteHeader(codeNum)

	acceptHeader := ex.HeaderValueLast("Accept")

	if acceptHeader == "application/json" {
		util.WriteJson(ex.ResponseWriter, map[string]interface{}{
			"code":        codeNum,
			"description": http.StatusText(codeNum),
		})

	} else if acceptHeader == "text/plain" {
		ex.WriteF("%d %s", codeNum, http.StatusText(codeNum))

	}
}

// Digest auth response computer.
func computeDigestAuthResponse(username, password, serverNonce, nc, clientNonce, qop, method, path string) string {
	// Source: <https://en.wikipedia.org/wiki/Digest_access_authentication>.
	ha1 := util.Md5sum(username + ":" + "testrealm@host.com" + ":" + password)
	ha2 := util.Md5sum(method + ":" + path)
	return util.Md5sum(ha1 + ":" + serverNonce + ":" + nc + ":" + clientNonce + ":" + qop + ":" + ha2)
}

func handleIp(ex *exchange.Exchange) {
	origin := ex.FindIncomingIPAddress()
	if ex.Field("format") == "txt" {
		ex.Write(origin)
	} else {
		util.WriteJson(ex.ResponseWriter, map[string]string{
			"origin": origin,
		})
	}
}

func handleUserAgent(ex *exchange.Exchange) {
	util.WriteJson(ex.ResponseWriter, map[string]string{
		"user-agent": ex.HeaderValueLast("User-Agent"),
	})
}

func handleCache(ex *exchange.Exchange) {
	shouldSendData :=
		ex.HeaderValueLast("If-Modified-Since") == "" &&
			ex.HeaderValueLast("If-None-Match") == ""

	if shouldSendData {
		isNonGet := ex.Request.Method != http.MethodGet
		sendInfoJson(ex, InfoJsonOptions{
			BodyInfo: isNonGet,
		})
	} else {
		ex.ResponseWriter.WriteHeader(http.StatusNotModified)
	}
}

func handleCacheControl(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("Cache-Control", "public, max-age="+ex.Field("age"))
	isNonGet := ex.Request.Method != http.MethodGet
	sendInfoJson(ex, InfoJsonOptions{
		BodyInfo: isNonGet,
	})
}

func handleEtag(ex *exchange.Exchange) {
	// TODO: Handle If-Match header in etag endpoint: <https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Match>.
	etagInUrl := ex.Field("etag")
	etagInHeader := ex.HeaderValueLast("If-None-Match")

	if etagInUrl == etagInHeader {
		ex.ResponseWriter.WriteHeader(http.StatusNotModified)
	} else {
		isNonGet := ex.Request.Method != http.MethodGet
		sendInfoJson(ex, InfoJsonOptions{
			BodyInfo: isNonGet,
		})
	}
}

func handleResponseHeaders(ex *exchange.Exchange) {
	data := make(map[string]interface{})

	for name, values := range ex.Request.URL.Query() {
		name = http.CanonicalHeaderKey(name)
		for _, value := range values {
			ex.ResponseWriter.Header().Add(name, value)
		}
		if len(values) > 1 {
			data[name] = values
		} else {
			data[name] = values[0]
		}
	}

	ex.ResponseWriter.Header().Set("Content-Type", "application/json")
	data["Content-Type"] = "application/json"

	var jsonContent []byte

	for {
		jsonContent = util.ToJsonMust(data)
		newContentLength := fmt.Sprint(len(jsonContent))
		if data["Content-Length"] == newContentLength {
			break
		}
		data["Content-Length"] = newContentLength
	}

	ex.WriteBytes(jsonContent)
}

func handleDecodeBase64(ex *exchange.Exchange) {
	encoded := ex.Field("encoded")
	if encoded == "" {
		encoded = "SFRUUEJVTiBpcyBhd2Vzb21lciE="
	}
	if decoded, err := base64.StdEncoding.DecodeString(encoded); err != nil {
		ex.Write("Incorrect Base64 data try: 'SFRUUEJVTiBpcyBhd2Vzb21lciE='.")
	} else {
		ex.WriteBytes(decoded)
	}
}

func handleRandomBytes(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("content-type", "application/octet-stream")
	n, _ := strconv.Atoi(ex.Field("size"))
	ex.WriteBytes(util.RandomBytes(n))
}

func handleDelayedResponse(ex *exchange.Exchange) {
	n, err := strconv.ParseFloat(ex.Field("delay"), 32)

	if err != nil {
		ex.RespondBadRequest("Invalid delay: " + err.Error())
		return
	}

	if n > 20 {
		ex.RespondBadRequest("Delay can't be greater than 20")
		return
	}

	time.Sleep(time.Duration(n * float64(time.Second)))
}

func handleDrip(ex *exchange.Exchange) {
	// Test with `curl -N localhost:3090/drip`.

	writeNewLines := ex.Field("mode") == "lines"

	duration, err := ex.QueryParamInt("duration", 2)
	if err != nil {
		ex.RespondBadRequest(err.Error())
		return
	}

	numbytes, err := ex.QueryParamInt("numbytes", 10)
	if err != nil {
		ex.RespondBadRequest(err.Error())
		return
	}

	code, err := ex.QueryParamInt("code", 200)
	if err != nil {
		ex.RespondBadRequest(err.Error())
		return
	}

	delay, err := ex.QueryParamInt("delay", 2)
	if err != nil {
		ex.RespondBadRequest(err.Error())
		return
	}

	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Second)
	}

	ex.ResponseWriter.Header().Set("Cache-Control", "no-cache")
	ex.ResponseWriter.Header().Set("Content-Type", "text/event-stream")
	ex.ResponseWriter.WriteHeader(code)

	interval := time.Duration(float32(time.Second) * float32(duration) / float32(numbytes))

	for numbytes > 0 {
		ex.Write("*")
		if writeNewLines {
			ex.Write("\n")
		}
		if !util.Flush(ex.ResponseWriter) {
			log.Println("Flush not available. Dripping and streaming not supported on this platform.")
		}
		time.Sleep(interval)
		numbytes--
	}
}

func handleLinks(ex *exchange.Exchange) {
	count, _ := strconv.Atoi(ex.Field("count"))
	offset, _ := strconv.Atoi(ex.Field("offset"))

	ex.Write("<html><head><title>Links</title></head><body>")
	for i := 0; i < count; i++ {
		if offset == i {
			ex.Write(strconv.Itoa(i))
		} else {
			ex.WriteF("<a href='/links/%d/%d'>%d</a>", count, i, i)
		}
		ex.Write(" ")
	}
	ex.Write("</body></html>")
}

func handleRange(ex *exchange.Exchange) {
	// TODO: Cache range response, don't have to generate over and over again.
	count, _ := strconv.Atoi(ex.Field("count"))

	if count > 1000 {
		count = 1000
	} else if count < 0 {
		count = 0
	}

	ex.ResponseWriter.Header().Set("content-type", "application/octet-stream")

	if count > 0 {
		r := rand.New(rand.NewSource(42))
		b := make([]byte, count)
		r.Read(b)
		ex.WriteBytes(b)
	}
}

func handleInfo(ex *exchange.Exchange) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "Error: " + err.Error()
	}

	util.WriteJson(ex.ResponseWriter, map[string]interface{}{
		"hostname": hostname,
	})
}

func handleMix(ex *exchange.Exchange) {
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

	actions := url.Values{}

	for _, part := range strings.Split(source, itemSep) {
		if part == "" {
			continue
		}
		key, value, _ := strings.Cut(part, "=")
		key, err := unescape(key)
		if err != nil {
			ex.RespondBadRequest(err.Error())
			return
		}
		value, err = unescape(value)
		if err != nil {
			ex.RespondBadRequest(err.Error())
			return
		}
		actions[key] = append(actions[key], value)
	}

	status := http.StatusOK
	headers := url.Values{}
	cookies := map[string]string{}
	var deleteCookies []string
	var payload []byte

	if actions.Has("h") {
		for _, headerSpec := range actions["h"] {
			name, value, _ := strings.Cut(headerSpec, ":")
			name = http.CanonicalHeaderKey(name)
			headers.Add(name, value)
		}
	}

	if actions.Has("c") {
		for _, headerSpec := range actions["c"] {
			name, value, isFound := strings.Cut(headerSpec, ":")
			if isFound {
				cookies[name] = value
			} else {
				deleteCookies = append(deleteCookies, name)
			}
		}
	}

	if actions.Has("r") {
		status = http.StatusTemporaryRedirect
		headers.Set("Location", actions.Get("r"))
	}

	if actions.Has("s") {
		var err error
		status, err = strconv.Atoi(actions.Get("s"))
		if err != nil {
			ex.RespondBadRequest(err.Error())
			return
		}
	}

	if actions.Has("d") {
		seconds, err := strconv.ParseFloat(actions.Get("d"), 32)
		if err != nil {
			ex.RespondBadRequest(err.Error())
			return
		}
		if seconds > 10 {
			ex.RespondBadRequest("Delay must be less than 10 seconds.")
			return
		}
		time.Sleep(time.Duration(seconds * float64(time.Second)))
	}

	if actions.Has("b64") {
		if decoded, err := base64.StdEncoding.DecodeString(actions.Get("b64")); err != nil {
			ex.RespondBadRequest("Incorrect Base64 data try: 'SFRUUEJVTiBpcyBhd2Vzb21lciE='.")
			return
		} else {
			payload = decoded
		}
	}

	for name, value := range headers {
		ex.ResponseWriter.Header()[name] = value
	}

	for name, value := range cookies {
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  name,
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
