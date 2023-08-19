package bun

import (
	"encoding/base64"
	"fmt"
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/bun/auth"
	"github.com/sharat87/httpbun/bun/mix"
	"github.com/sharat87/httpbun/bun/redirect"
	"github.com/sharat87/httpbun/bun/responses"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/mux"
	"github.com/sharat87/httpbun/server/spec"
	"github.com/sharat87/httpbun/util"
	"io"
	"log"
	"maps"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func MakeBunHandler(spec spec.Spec) mux.Mux {
	m := mux.Mux{
		ServerSpec: spec,
	}

	m.HandleFunc(`(/(index\.html)?)?`, handleIndex)

	m.HandleFunc(`/(?P<name>.+\.(png|ico|webmanifest))`, func(ex *exchange.Exchange) {
		assets.WriteAsset(ex.Field("name"), *ex)
	})

	m.HandleFunc("/health", handleHealth)

	m.HandleFunc("/get", handleValidMethod)
	m.HandleFunc("/post", handleValidMethod)
	m.HandleFunc("/put", handleValidMethod)
	m.HandleFunc("/patch", handleValidMethod)
	m.HandleFunc("/delete", handleValidMethod)

	m.HandleFunc("/headers", handleHeaders)

	m.HandleFunc("/payload", handlePayload)

	m.HandleFunc("/status/(?P<codes>[\\d,]+)", handleStatus)
	m.HandleFunc("/ip(\\.(?P<format>txt|json))?", handleIp)

	m.HandleFunc("/cache", handleCache)
	m.HandleFunc("/cache/(?P<age>\\d+)", handleCacheControl)
	m.HandleFunc("/etag/(?P<etag>[^/]+)", handleEtag)
	m.HandleFunc("/(response|respond-with)-headers?/?", handleResponseHeaders)

	m.HandleFunc("/deny", handleSampleRobotsDeny)
	m.HandleFunc("/html", handleSampleHtml)
	m.HandleFunc("/robots.txt", handleSampleRobotsTxt)
	m.HandleFunc("/image/svg", handleImageSvg)

	m.HandleFunc("/b(ase)?64(/(?P<encoded>.*))?", handleDecodeBase64)
	m.HandleFunc("/bytes/(?P<size>\\d+)", handleRandomBytes)
	m.HandleFunc("/delay/(?P<delay>[^/]+)", handleDelayedResponse)
	m.HandleFunc("/drip(-(?P<mode>lines))?", handleDrip)
	m.HandleFunc("/links/(?P<count>\\d+)(/(?P<offset>\\d+))?/?", handleLinks)
	m.HandleFunc("/range/(?P<count>\\d+)/?", handleRange)

	m.HandleFunc("/cookies", handleCookies)
	m.HandleFunc("/cookies/delete", handleCookiesDelete)
	m.HandleFunc("/cookies/set(/(?P<name>[^/]+)/(?P<value>[^/]+))?", handleCookiesSet)

	m.HandleFunc("/any(thing)?\\b.*", handleAnything)

	m.HandleFunc("/info", handleInfo)

	m.HandleFunc("/(?P<hook>hooks.slack.com/services/.*)", handleSlack)

	allRoutes := map[string]mux.HandlerFn{}

	maps.Copy(allRoutes, auth.Routes)
	maps.Copy(allRoutes, redirect.Routes)
	maps.Copy(allRoutes, mix.Routes)

	for pat, fn := range allRoutes {
		m.HandleFunc(pat, fn)
	}

	return m
}

func handleIndex(ex *exchange.Exchange) {
	assets.Render("index.html", *ex, map[string]any{
		"host": ex.URL.Host,
	})
}

func handleHealth(ex *exchange.Exchange) {
	ex.WriteLn("ok")
}

func handleValidMethod(ex *exchange.Exchange) {
	allowedMethod := strings.ToUpper(strings.TrimPrefix(ex.URL.Path, "/"))
	if ex.Request.Method != allowedMethod {
		allowedMethods := allowedMethod + ", " + http.MethodOptions
		ex.ResponseWriter.Header().Set("Allow", allowedMethods)
		ex.ResponseWriter.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		if ex.Request.Method != http.MethodOptions {
			ex.ResponseWriter.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	responses.InfoJSON(ex)
}

func handleAnything(ex *exchange.Exchange) {
	responses.InfoJSON(ex)
}

func handleHeaders(ex *exchange.Exchange) {
	util.WriteJson(ex.ResponseWriter, ex.ExposableHeadersMap())
}

func handlePayload(ex *exchange.Exchange) {
	ex.ResponseWriter.Header()[c.ContentType] = ex.Request.Header[c.ContentType]
	_, err := io.Copy(ex.ResponseWriter, ex.CappedBody)
	if err != nil {
		fmt.Println("Error reading request payload", err)
	}
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

	if acceptHeader == c.ApplicationJSON {
		util.WriteJson(ex.ResponseWriter, map[string]any{
			"code":        codeNum,
			"description": http.StatusText(codeNum),
		})

	} else if acceptHeader == c.TextPlain {
		ex.WriteF("%d %s", codeNum, http.StatusText(codeNum))

	}
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

func handleCache(ex *exchange.Exchange) {
	shouldSendData :=
		ex.HeaderValueLast("If-Modified-Since") == "" &&
			ex.HeaderValueLast("If-None-Match") == ""

	if shouldSendData {
		responses.InfoJSON(ex)
	} else {
		ex.ResponseWriter.WriteHeader(http.StatusNotModified)
	}
}

func handleCacheControl(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("Cache-Control", "public, max-age="+ex.Field("age"))
	responses.InfoJSON(ex)
}

func handleEtag(ex *exchange.Exchange) {
	// TODO: Handle If-Match header in etag endpoint: <https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Match>.
	etagInUrl := ex.Field("etag")
	etagInHeader := ex.HeaderValueLast("If-None-Match")

	if etagInUrl == etagInHeader {
		ex.ResponseWriter.WriteHeader(http.StatusNotModified)
	} else {
		responses.InfoJSON(ex)
	}
}

func handleResponseHeaders(ex *exchange.Exchange) {
	data := make(map[string]any)

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

	ex.ResponseWriter.Header().Set(c.ContentType, c.ApplicationJSON)
	data[c.ContentType] = c.ApplicationJSON

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

	if n > 300 {
		ex.RespondBadRequest("Delay can't be greater than 300")
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
	ex.ResponseWriter.Header().Set(c.ContentType, "text/event-stream")
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

	env := make(map[string]any)
	for _, e := range os.Environ() {
		name, value, _ := strings.Cut(e, "=")
		// These env variables get auto-set when run in Docker, so we set marker values in the _image_, and if they're
		// not set for the _container_, we'll just not include them in the output.
		if value != "___httpbun_unset_marker" || (name != "PATH" && name != "HOME" && name != "HOSTNAME") {
			env[name] = value
		}
	}

	util.WriteJson(ex.ResponseWriter, map[string]any{
		"hostname": hostname,
		"env":      env,
	})
}
