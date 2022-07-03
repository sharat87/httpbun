package bun

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/mux"
	"github.com/sharat87/httpbun/storage"
	"github.com/sharat87/httpbun/util"
	"html/template"
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

func MakeBunHandler(pathPrefix, database string) mux.Mux {
	var st storage.Storage

	if database == "" {
		database = "sqlite://httpbun.db?mode=memory"
	}

	if strings.HasPrefix(database, "sqlite://") {
		st = storage.NewSqliteStorage(strings.TrimPrefix(database, "sqlite://"))
	} else if strings.HasPrefix(database, "mongodb+srv://") || strings.HasPrefix(database, "mongodb://") {
		st = storage.NewMongoStorage(database)
	} else {
		log.Fatalf("Unsupported database: %q", database)
	}

	m := mux.Mux{
		PathPrefix: pathPrefix,
		Storage:    st,
	}

	m.HandleFunc(`/(index\.html)?`, func(ex *exchange.Exchange) {
		ex.ResponseWriter.Header().Set("Content-Type", "text/html")
		assets.Render("index.html", ex.ResponseWriter, map[string]string{
			"Host": ex.URL.Host,
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

	m.HandleFunc("/basic-auth/(?P<user>[^/]+)/(?P<pass>[^/]+)/?", handleAuthBasic)
	m.HandleFunc("/bearer(/(?P<tok>\\w+))?", handleAuthBearer)
	m.HandleFunc("/digest-auth/(?P<qop>[^/]+)/(?P<user>[^/]+)/(?P<pass>[^/]+)/?", handleAuthDigest)

	m.HandleFunc("/status/(?P<codes>[\\d,]+)", handleStatus)
	m.HandleFunc("/ip", handleIp)
	m.HandleFunc("/user-agent", handleUserAgent)

	m.HandleFunc("/cache", handleCache)
	m.HandleFunc("/cache/(?P<age>\\d+)", handleCacheControl)
	m.HandleFunc("/etag/(?P<etag>[^/]+)", handleEtag)
	m.HandleFunc("/response-headers", handleResponseHeaders)

	m.HandleFunc("/deny", handleSampleRobotsDeny)
	m.HandleFunc("/html", handleSampleHtml)
	m.HandleFunc("/json", handleSampleJson)
	m.HandleFunc("/robots.txt", handleSampleRobotsTxt)
	m.HandleFunc("/xml", handleSampleXml)
	m.HandleFunc("/image/svg1", handleImageSvg1)

	m.HandleFunc("/base64(/(?P<encoded>.*))?", handleDecodeBase64)
	m.HandleFunc("/bytes/(?P<size>\\d+)", handleRandomBytes)
	m.HandleFunc("/delay/(?P<delay>\\d+)", handleDelayedResponse)
	m.HandleFunc("/drip(-(?P<mode>lines))?", handleDrip)
	m.HandleFunc("/links/(?P<count>\\d+)(/(?P<offset>\\d+))?/?", handleLinks)
	m.HandleFunc("/range/(?P<count>\\d+)/?", handleRange)

	m.HandleFunc("/cookies", handleCookies)
	m.HandleFunc("/cookies/delete", handleCookiesDelete)
	m.HandleFunc("/cookies/set(/(?P<name>[^/]+)/(?P<value>[^/]+))?", handleCookiesSet)

	m.HandleFunc("/redirect-to/?", handleRedirectTo)
	m.HandleFunc("/(relative-)?redirect/(?P<count>\\d+)", handleRelativeRedirect)
	m.HandleFunc("/absolute-redirect/(?P<count>\\d+)", handleAbsoluteRedirect)

	m.HandleFunc("/anything\\b.*", handleAnything)

	m.HandleFunc("/iframe", handleFrame)

	m.HandleFunc("/oauth/authorize", handleOauthAuthorize)
	m.HandleFunc("/oauth/authorize/submit", handleOauthAuthorizeSubmit)

	const inboxPat = "/inbox/(?P<name>[-_a-z0-9]+?)"
	m.HandleFunc(inboxPat, handleInboxPush)
	m.HandleFunc(inboxPat+"/view", handleInboxView)

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
		ex.ResponseWriter.Header().Set("Allow", allowedMethod)
		ex.ResponseWriter.WriteHeader(http.StatusMethodNotAllowed)
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
		"origin":  ex.FindOrigin(),
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
		form := make(map[string]interface{})
		var jsonData *interface{}
		files := make(map[string]interface{})
		data := ""

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
				if f, err := fileHeader.Open(); err != nil {
					fmt.Println("Error opening fileHeader", err)
				} else if content, err := ioutil.ReadAll(f); err != nil {
					fmt.Println("Error reading from fileHeader", err)
				} else if utf8.Valid(content) {
					files[name] = string(content)
				} else {
					files[name] = content
				}
			}

			for name, valueInfo := range allFileData.Value {
				form[name] = valueInfo[0]
			}

		} else {
			data = ex.BodyString()

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

	acceptHeader := ex.HeaderValueLast("accept")

	if acceptHeader == "application/json" {
		util.WriteJson(ex.ResponseWriter, map[string]interface{}{
			"code":        codeNum,
			"description": http.StatusText(codeNum),
		})

	} else {
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
	util.WriteJson(ex.ResponseWriter, map[string]string{
		"origin": ex.FindOrigin(),
	})
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

	jsonContent := ""

	for {
		jsonContent = util.ToJsonMust(data)
		newContentLength := fmt.Sprint(len(jsonContent))
		if data["Content-Length"] == newContentLength {
			break
		}
		data["Content-Length"] = newContentLength
	}

	ex.WriteLn(jsonContent)
}

func handleDecodeBase64(ex *exchange.Exchange) {
	encoded := ex.Field("encoded")
	if encoded == "" {
		encoded = "SFRUUEJVTiBpcyBhd2Vzb21lciE="
	}
	if decoded, err := base64.StdEncoding.DecodeString(encoded); err != nil {
		ex.Write("Incorrect Base64 data try: 'SFRUUEJVTiBpcyBhd2Vzb21lciE='.")
	} else {
		ex.Write(string(decoded))
	}
}

func handleRandomBytes(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("content-type", "application/octet-stream")
	n, _ := strconv.Atoi(ex.Field("size"))
	ex.Write(util.RandomBytes(n))
}

func handleDelayedResponse(ex *exchange.Exchange) {
	n, _ := strconv.Atoi(ex.Field("delay"))
	time.Sleep(time.Duration(n) * time.Second)
}

func handleDrip(ex *exchange.Exchange) {
	// Test with `curl -N localhost:3090/drip`.

	writeNewLines := ex.Field("mode") == "lines"

	duration, err := ex.QueryParamInt("duration", 2)
	if err != nil {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ex.WriteLn(err.Error())
		return
	}

	numbytes, err := ex.QueryParamInt("numbytes", 10)
	if err != nil {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ex.WriteLn(err.Error())
		return
	}

	code, err := ex.QueryParamInt("code", 200)
	if err != nil {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ex.WriteLn(err.Error())
		return
	}

	delay, err := ex.QueryParamInt("delay", 2)
	if err != nil {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ex.WriteLn(err.Error())
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
			ex.Write(i)
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
		ex.Write(b)
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

func handleInboxPush(ex *exchange.Exchange) {
	inboxName := ex.Field("name")
	if len(inboxName) > 80 {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ex.WriteLn("Inbox name too long. Max is 80 characters.")
		return
	}

	// Respond immediately, and have the request saved in a separate thread of execution.
	go ex.Storage.PushRequestToInbox(inboxName, *ex.Request)
	ex.RespondWithStatus(http.StatusOK)
}

func handleInboxView(ex *exchange.Exchange) {
	name := ex.Field("name")
	entries := ex.Storage.GetFromInbox(name)
	assets.Render("inbox-view.html", ex.ResponseWriter, template.JS(util.ToJsonMust(map[string]interface{}{
		"name":    name,
		"entries": entries,
	})))
}
