package bun

import (
	"encoding/base64"
	"fmt"
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/bun/auth"
	"github.com/sharat87/httpbun/bun/cache"
	"github.com/sharat87/httpbun/bun/headers"
	"github.com/sharat87/httpbun/bun/method"
	"github.com/sharat87/httpbun/bun/mix"
	"github.com/sharat87/httpbun/bun/redirect"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
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

type handlerFn func(ex *exchange.Exchange)
type Route struct {
	Pat regexp.Regexp
	Fn  handlerFn
}

func GetRoutes() []Route {
	var routes []Route

	addRoute := func(pattern string, fn func(ex *exchange.Exchange)) {
		routes = append(routes, Route{
			Pat: *regexp.MustCompile("^" + pattern + "$"),
			Fn:  fn,
		})
	}

	addRoute(`(/(index\.html)?)?`, handleIndex)

	addRoute(`/(?P<name>.+\.(png|ico|webmanifest))`, func(ex *exchange.Exchange) {
		assets.WriteAsset(ex.Field("name"), *ex)
	})

	addRoute("/health", handleHealth)

	addRoute("/payload", handlePayload)

	addRoute("/status/(?P<codes>[\\d,]+)", handleStatus)
	addRoute("/ip(\\.(?P<format>txt|json))?", handleIp)

	addRoute("/deny", handleSampleRobotsDeny)
	addRoute("/html", handleSampleHtml)
	addRoute("/robots.txt", handleSampleRobotsTxt)
	addRoute("/image/svg", handleImageSvg)

	addRoute("/b(ase)?64(/(?P<encoded>.*))?", handleDecodeBase64)
	addRoute("/bytes/(?P<size>\\d+)", handleRandomBytes)
	addRoute("/delay/(?P<delay>[^/]+)", handleDelayedResponse)
	addRoute("/drip(-(?P<mode>lines))?", handleDrip)
	addRoute("/links/(?P<count>\\d+)(/(?P<offset>\\d+))?/?", handleLinks)
	addRoute("/range/(?P<count>\\d+)/?", handleRange)

	addRoute("/cookies", handleCookies)
	addRoute("/cookies/delete", handleCookiesDelete)
	addRoute("/cookies/set(/(?P<name>[^/]+)/(?P<value>[^/]+))?", handleCookiesSet)

	addRoute("/info", handleInfo)

	addRoute("/(?P<hook>hooks.slack.com/services/.*)", handleSlack)

	allRoutes := map[string]exchange.HandlerFn{}

	maps.Copy(allRoutes, method.Routes)
	maps.Copy(allRoutes, headers.Routes)
	maps.Copy(allRoutes, cache.Routes)
	maps.Copy(allRoutes, auth.Routes)
	maps.Copy(allRoutes, redirect.Routes)
	maps.Copy(allRoutes, mix.Routes)

	for pat, fn := range allRoutes {
		addRoute(pat, fn)
	}

	return routes
}

func handleIndex(ex *exchange.Exchange) {
	assets.Render("index.html", *ex, map[string]any{
		"host": ex.URL.Host,
	})
}

func handleHealth(ex *exchange.Exchange) {
	ex.WriteLn("ok")
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
	ex.ResponseWriter.Header().Set("content-length", fmt.Sprint(n))
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
