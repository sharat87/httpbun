package routes

import (
	"encoding/base64"
	"fmt"
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"github.com/sharat87/httpbun/routes/auth"
	"github.com/sharat87/httpbun/routes/cache"
	"github.com/sharat87/httpbun/routes/cookies"
	"github.com/sharat87/httpbun/routes/headers"
	"github.com/sharat87/httpbun/routes/method"
	"github.com/sharat87/httpbun/routes/mix"
	"github.com/sharat87/httpbun/routes/redirect"
	"github.com/sharat87/httpbun/routes/run"
	"github.com/sharat87/httpbun/routes/sse"
	"github.com/sharat87/httpbun/routes/static"
	"github.com/sharat87/httpbun/util"
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

type Route struct {
	Pat regexp.Regexp
	Fn  exchange.HandlerFn
}

func GetRoutes() []Route {
	var routes []Route

	allRoutes := map[string]exchange.HandlerFn{
		"/health": handleHealth,

		"/b(ase)?64(/(?P<encoded>.*))?":                handleDecodeBase64,
		"/bytes(/(?P<size>.+))?":                       handleRandomBytes,
		"/drip(-(?P<mode>lines))?(?P<extra>/.*)?":      handleDrip,
		"/links/(?P<count>\\d+)(/(?P<offset>\\d+))?/?": handleLinks,
		"/range/(?P<count>\\d+)/?":                     handleRange,

		"/info": handleInfo,

		"/(?P<hook>hooks.slack.com/services/.*)": handleSlack,
	}

	allRoutes2 := map[string]exchange.HandlerFn2{
		`/assets/(?P<path>.+)`: handleAsset,

		`(/(index\.html)?)?`: handleIndex,

		"/delay/(?P<delay>[^/]+)": handleDelayedResponse,

		"/payload": handlePayload,

		"/status/(?P<codes>[\\w,]+)": handleStatus,

		"/ip(\\.(?P<format>txt|json))?": handleIp,
	}

	maps.Copy(allRoutes2, method.Routes)
	maps.Copy(allRoutes, headers.Routes)
	maps.Copy(allRoutes2, cache.Routes)
	maps.Copy(allRoutes2, auth.Routes)
	maps.Copy(allRoutes, redirect.Routes)
	maps.Copy(allRoutes2, mix.Routes)
	maps.Copy(allRoutes2, static.Routes)
	maps.Copy(allRoutes, cookies.Routes)
	maps.Copy(allRoutes2, run.Routes)
	maps.Copy(allRoutes, sse.Routes)

	for pat, fn := range allRoutes {
		routes = append(routes, Route{
			Pat: *regexp.MustCompile("^" + pat + "$"),
			Fn:  fn,
		})
	}

	for pat, fn := range allRoutes2 {
		routes = append(routes, Route{
			Pat: *regexp.MustCompile("(?s)^" + pat + "$"),
			Fn: (func(fn exchange.HandlerFn2) exchange.HandlerFn {
				return func(ex *exchange.Exchange) {
					ex.Finish(fn(ex))
				}
			})(fn),
		})
	}

	return routes
}

func handleIndex(ex *exchange.Exchange) response.Response {
	return assets.Render("index.html", *ex, nil)
}

func handleAsset(ex *exchange.Exchange) response.Response {
	path := ex.Field("path")
	if strings.Contains(path, "..") {
		ex.RespondBadRequest("Assets path cannot contain '..'.")
	}
	return assets.WriteAsset(path)
}

func handleHealth(ex *exchange.Exchange) {
	ex.WriteLn("ok")
}

func handlePayload(ex *exchange.Exchange) response.Response {
	return response.New(http.StatusOK, http.Header{
		c.ContentType: ex.Request.Header[c.ContentType],
	}, ex.BodyBytes())
}

func handleStatus(ex *exchange.Exchange) response.Response {
	input := ex.Field("codes")
	if len(input) > 99 {
		return response.BadRequest("Too many status codes")
	}

	parts := strings.Split(input, ",")
	var codes []int

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		code, err := strconv.Atoi(part)
		if err != nil {
			return response.BadRequest("Invalid status code: " + part)
		}
		if code < 100 || code > 599 {
			return response.BadRequest("Invalid status code: " + part)
		}
		codes = append(codes, code)
	}

	var status int
	if len(codes) > 1 {
		status = codes[rand.Intn(len(codes))]
	} else {
		status = codes[0]
	}

	acceptHeader := ex.HeaderValueLast("Accept")

	if strings.HasPrefix(acceptHeader, c.TextPlain) {
		return response.New(status, nil, []byte(http.StatusText(status)))

	} else {
		return response.JSON(status, nil, map[string]any{
			"code":        status,
			"description": http.StatusText(status),
		})

	}
}

func handleIp(ex *exchange.Exchange) response.Response {
	origin := ex.FindIncomingIPAddress()
	if ex.Field("format") == "txt" {
		ex.Write(origin)
		return response.New(http.StatusOK, nil, []byte(origin))
	} else {
		return response.JSON(http.StatusOK, nil, map[string]any{
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
	sizeField := ex.Field("size")
	if sizeField == "" {
		ex.RespondBadRequest("specify size in bytes, example `/bytes/10`")
		return
	}
	n, err := strconv.Atoi(sizeField)
	if err != nil {
		ex.RespondBadRequest("Invalid size: " + sizeField)
		return
	}
	ex.ResponseWriter.Header().Set("content-type", "application/octet-stream")
	ex.ResponseWriter.Header().Set("content-length", fmt.Sprint(n))
	ex.WriteBytes(util.RandomBytes(n))
}

func handleDelayedResponse(ex *exchange.Exchange) response.Response {
	n, err := strconv.ParseFloat(ex.Field("delay"), 32)

	if err != nil {
		return response.BadRequest("Invalid delay: " + err.Error())
	}

	if n < 0 || n > 300 {
		return response.BadRequest("Delay can't be greater than 300 or less than 0")
	}

	time.Sleep(time.Duration(n * float64(time.Second)))
	return response.New(http.StatusOK, nil, []byte("OK"))
}

func handleDrip(ex *exchange.Exchange) {
	// Test with `curl -N localhost:3090/drip`.

	extra := ex.Field("extra")
	if extra != "" {
		// todo: docs duplicated from index.html
		ex.RespondBadRequest("Unknown extra path: " + extra +
			"\nUse `/drip` or `/drip-lines` with query params:\n" +
			"  duration: Total number of seconds over which to stream the data. Default: 2.\n" +
			"  numbytes: Total number of bytes to stream. Default: 10.\n" +
			"  code: The HTTP status code to be used in their response. Default: 200.\n" +
			"  delay: An initial delay, in seconds. Default: 2.\n",
		)
		return
	}

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
	ex.ResponseWriter.Header().Set(c.ContentType, "text/octet-stream")
	ex.ResponseWriter.WriteHeader(code)

	interval := time.Duration(float32(time.Second) * float32(duration) / float32(numbytes))

	for numbytes > 0 {
		ex.Write("*")
		if writeNewLines {
			ex.Write("\n")
		}
		f, ok := ex.ResponseWriter.(http.Flusher)
		if ok {
			f.Flush()
		} else {
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

	ex.WriteJSON(map[string]any{
		"hostname": hostname,
		"env":      env,
	})
}
