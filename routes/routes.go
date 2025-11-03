package routes

import (
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/ex"
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
	"github.com/sharat87/httpbun/routes/svg"
	"github.com/sharat87/httpbun/util"
)

func GetRoutes() []ex.Route {
	return slices.Concat(
		[]ex.Route{
			ex.NewRoute("/health", handleHealth),
			ex.NewRoute("/info", handleInfo),

			ex.NewRoute("/b(ase)?64(/(?P<encoded>.*))?", handleDecodeBase64),
			ex.NewRoute("/bytes(/(?P<size>.+))?", handleRandomBytes),
			ex.NewRoute("/links/(?P<count>\\d+)(/(?P<offset>\\d+))?/?", handleLinks),
			ex.NewRoute("/range/(?P<count>\\d+)/?", handleRange),

			ex.NewRoute("/drip(-(?P<mode>lines))?(?P<extra>/.*)?", handleDrip),

			ex.NewRoute(`/assets/(?P<path>.+)`, handleAsset),
			ex.NewRoute(`(/(index\.html)?)?`, handleIndex),

			ex.NewRoute("/delay/(?P<delay>[^/]+)", handleDelayedResponse),

			ex.NewRoute("/payload", handlePayload),
			ex.NewRoute("/status/(?P<codes>[\\w,]+)", handleStatus),
			ex.NewRoute("/ip(\\.(?P<format>txt|json))?", handleIp),
		},
		auth.RouteList,
		cache.RouteList,
		cookies.RouteList,
		headers.RouteList,
		method.RouteList,
		mix.RouteList,
		redirect.RouteList,
		run.RouteList,
		sse.RouteList,
		static.RouteList,
		svg.RouteList,
	)
}

func handleIndex(ex *ex.Exchange) response.Response {
	return assets.Render("index.html", *ex, nil)
}

func handleAsset(ex *ex.Exchange) response.Response {
	path := ex.Field("path")
	if strings.Contains(path, "..") {
		return response.BadRequest("Assets path cannot contain '..'.")
	}
	return *assets.WriteAsset(path)
}

func handleHealth(_ *ex.Exchange) response.Response {
	return response.Response{Body: "ok"}
}

func handlePayload(ex *ex.Exchange) response.Response {
	return response.New(http.StatusOK, http.Header{
		c.ContentType: ex.Request.Header[c.ContentType],
	}, ex.BodyBytes())
}

func handleStatus(ex *ex.Exchange) response.Response {
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
			return response.BadRequest("Invalid status code: %s", part)
		}
		if code < 100 || code > 599 {
			return response.BadRequest("Invalid status code: %s", part)
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
		return response.Response{
			Status: status,
			Body: map[string]any{
				"code":        status,
				"description": http.StatusText(status),
			},
		}

	}
}

func handleIp(ex *ex.Exchange) response.Response {
	origin := ex.FindIncomingIPAddress()
	if ex.Field("format") == "txt" {
		return response.New(http.StatusOK, nil, []byte(origin))
	} else {
		return response.Response{
			Status: http.StatusOK,
			Body: map[string]any{
				"origin": origin,
			},
		}
	}
}

func handleDecodeBase64(ex *ex.Exchange) response.Response {
	encoded := ex.Field("encoded")
	if encoded == "" {
		encoded = "SFRUUEJVTiBpcyBhd2Vzb21lciE="
	}

	if decoded, err := base64.StdEncoding.DecodeString(encoded); err != nil {
		return response.BadRequest("Incorrect Base64 data try: 'SFRUUEJVTiBpcyBhd2Vzb21lciE='.")
	} else {
		return response.Response{
			Body: decoded,
		}
	}
}

func handleRandomBytes(ex *ex.Exchange) response.Response {
	sizeField := ex.Field("size")
	if sizeField == "" {
		return response.BadRequest("specify size in bytes, example `/bytes/10`")
	}

	n, err := strconv.Atoi(sizeField)
	if err != nil {
		return response.BadRequest("Invalid size: %s", sizeField)
	}

	if n > 90 {
		return response.BadRequest("Size can't be greater than 90")
	}

	return response.Response{
		Header: http.Header{
			c.ContentType:   []string{"application/octet-stream"},
			c.ContentLength: []string{fmt.Sprint(n)},
		},
		Body: util.RandomBytes(n),
	}
}

func handleDelayedResponse(ex *ex.Exchange) response.Response {
	n, err := strconv.ParseFloat(ex.Field("delay"), 32)

	if err != nil {
		return response.BadRequest("Invalid delay: %s", err.Error())
	}

	if n < 0 || n > 300 {
		return response.BadRequest("Delay can't be greater than 300 or less than 0")
	}

	time.Sleep(time.Duration(n * float64(time.Second)))
	return response.New(http.StatusOK, nil, []byte("OK"))
}

func handleDrip(ex *ex.Exchange) response.Response {
	// Test with `curl -N localhost:3090/drip`.

	extra := ex.Field("extra")
	if extra != "" {
		// todo: docs duplicated from index.html
		return response.BadRequest("Unknown extra path: %s" +
			"\nUse `/drip` or `/drip-lines` with query params:\n" +
			"  duration: Total number of seconds over which to stream the data. Default: 2.\n" +
			"  numbytes: Total number of bytes to stream. Default: 10.\n" +
			"  code: The HTTP status code to be used in their response. Default: 200.\n" +
			"  delay: An initial delay, in seconds. Default: 2.\n",
			extra,
		)
	}

	writeNewLines := ex.Field("mode") == "lines"

	duration, err := ex.QueryParamInt("duration", 2)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	numbytes, err := ex.QueryParamInt("numbytes", 10)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	code, err := ex.QueryParamInt("code", 200)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	delay, err := ex.QueryParamInt("delay", 2)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Second)
	}

	interval := time.Duration(float32(time.Second) * float32(duration) / float32(numbytes))

	return response.Response{
		Status: code,
		Header: http.Header{
			"Cache-Control": {"no-cache"},
			c.ContentType:   {"text/octet-stream"},
		},
		Writer: func(w response.BodyWriter) {
			for numbytes > 0 {
				part := "*"
				if writeNewLines {
					part += "\n"
				}
				err := w.Write(part)
				if err != nil {
					log.Printf("Error writing drip part: %v\n", err)
					return
				}
				time.Sleep(interval)
				numbytes--
			}
		},
	}
}

func handleLinks(ex *ex.Exchange) response.Response {
	count, _ := strconv.Atoi(ex.Field("count"))
	offset, _ := strconv.Atoi(ex.Field("offset"))

	var parts []string

	parts = append(parts, "<html><head><title>Links</title></head><body>")
	for i := 0; i < count; i++ {
		if offset == i {
			parts = append(parts, strconv.Itoa(i))
		} else {
			parts = append(parts, fmt.Sprintf("<a href='/links/%d/%d'>%d</a>", count, i, i))
		}
		parts = append(parts, " ")
	}
	parts = append(parts, "</body></html>")

	return response.Response{
		Body: strings.Join(parts, ""),
	}
}

func handleRange(ex *ex.Exchange) response.Response {
	// TODO: Cache range response, don't have to generate over and over again.
	count, _ := strconv.Atoi(ex.Field("count"))

	if count > 1000 {
		count = 1000
	} else if count < 0 {
		count = 0
	}

	var b []byte
	if count > 0 {
		b = make([]byte, count)
		rand.New(rand.NewSource(42)).Read(b)
	}

	return response.Response{
		Header: http.Header{
			c.ContentType: []string{"application/octet-stream"},
		},
		Body: b,
	}
}

func handleInfo(_ *ex.Exchange) response.Response {
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

	return response.Response{
		Body: map[string]any{
			"hostname": hostname,
			"env":      env,
		},
	}
}
