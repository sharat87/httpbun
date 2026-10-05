package routes

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"regexp"
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
	"github.com/sharat87/httpbun/routes/llm"
	"github.com/sharat87/httpbun/routes/method"
	"github.com/sharat87/httpbun/routes/mix"
	"github.com/sharat87/httpbun/routes/oauth2"
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
		oauth2.RouteList,
		redirect.RouteList,
		run.RouteList,
		sse.RouteList,
		static.RouteList,
		svg.RouteList,
		llm.RouteList,
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
	}, ex.BodyBytes(),
	)
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
		code, err := util.ParseStatusCode(part)
		if err != nil {
			return response.BadRequest("%s", err.Error())
		}
		codes = append(codes, code)
	}

	if len(codes) == 0 {
		return response.BadRequest("No status codes given")
	}

	var status int
	if len(codes) > 1 {
		status = codes[rand.Intn(len(codes))]
	} else {
		status = codes[0]
	}

	acceptHeader := ex.HeaderValueLast("Accept")

	if strings.HasPrefix(acceptHeader, "text/plain") {
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
	if ex.HadTrailingSlash && encoded != "" {
		// A trailing `/` can be part of standard base64 data.
		encoded += "/"
	}
	if encoded == "" {
		encoded = "SFRUUEJVTiBpcyBhd2Vzb21lciE="
	}

	// Accept both the standard and URL-safe alphabets, with or without padding.
	normalized := strings.TrimRight(strings.NewReplacer("-", "+", "_", "/").Replace(encoded), "=")
	if decoded, err := base64.RawStdEncoding.DecodeString(normalized); err != nil {
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

	n, err := ex.FieldInt("size", 0, ex.ServerSpec.EndpointBytesSizeLimit)
	if err != nil {
		return response.BadRequest("%s", err.Error())
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

	if math.IsNaN(n) || n < 0 || n > 300 {
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
		return response.BadRequest("Unknown extra path: %s"+
			"\nUse `/drip` or `/drip-lines` with query params:\n"+
			"  duration: Total number of seconds over which to stream the data, up to two decimal places. Default: 2.\n"+
			"  numbytes: Total number of bytes to stream. Default: 10.\n"+
			"  code: The HTTP status code to be used in their response. Default: 200.\n"+
			"  delay: An initial delay, in seconds, up to two decimal places. Default: 2.\n",
			extra,
		)
	}

	writeNewLines := ex.Field("mode") == "lines"

	duration, err := querySeconds(ex, "duration", 2*time.Second)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	numbytes, err := ex.QueryInt("numbytes", 10, 0, 10*1024*1024)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	code, err := ex.QueryInt("code", 200, 200, 599)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	delay, err := querySeconds(ex, "delay", 2*time.Second)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	if delay > 0 {
		time.Sleep(delay)
	}

	var interval time.Duration
	if numbytes > 0 {
		interval = duration / time.Duration(numbytes)
	}

	return response.Response{
		Status: code,
		Header: http.Header{
			"Cache-Control": {"no-cache"},
			c.ContentType:   {"application/octet-stream"},
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
				select {
				case <-ex.Request.Context().Done():
					// The client has disconnected.
					return
				case <-time.After(interval):
				}
				numbytes--
			}
		},
	}
}

var secondsPattern = regexp.MustCompile(`^(\d+(\.\d{1,2})?|\.\d{1,2})$`)

// querySeconds reads a query param as a non-negative number of seconds, with up to two decimal places.
func querySeconds(ex *ex.Exchange, name string, value time.Duration) (time.Duration, error) {
	values := ex.Request.URL.Query()[name]
	if len(values) == 0 {
		return value, nil
	}
	if !secondsPattern.MatchString(values[0]) {
		return 0, fmt.Errorf("%s must be a non-negative number of seconds, with up to two decimal places", name)
	}
	seconds, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number: %w", name, err)
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

func handleLinks(ex *ex.Exchange) response.Response {
	count, _ := strconv.Atoi(ex.Field("count"))
	offset, _ := strconv.Atoi(ex.Field("offset"))

	if count > 200 {
		count = 200
	}

	var parts []string

	parts = append(parts, "<html><head><title>Links</title></head><body>")
	for i := 0; i < count; i++ {
		if offset == i {
			parts = append(parts, strconv.Itoa(i))
		} else {
			parts = append(parts, fmt.Sprintf("<a href='%s/links/%d/%d'>%d</a>", ex.ServerSpec.PathPrefix, count, i, i))
		}
		parts = append(parts, " ")
	}
	parts = append(parts, "</body></html>")

	return response.Response{
		Body: strings.Join(parts, ""),
	}
}

func handleRange(ex *ex.Exchange) response.Response {
	count, _ := strconv.Atoi(ex.Field("count"))

	if count > 1000 {
		count = 1000
	} else if count < 0 {
		count = 0
	}

	// Each byte is its own offset (mod 256), so it's easy to check a partial response starts at the right place.
	b := make([]byte, count)
	for i := range b {
		b[i] = byte(i % 256)
	}

	header := http.Header{
		c.ContentType:   []string{"application/octet-stream"},
		"Accept-Ranges": []string{"bytes"},
	}

	start, end, ok := parseByteRange(ex.HeaderValueLast("Range"), count)
	if !ok {
		// No range, or one we don't support, like multiple ranges. Servers can ignore those, and send everything.
		return response.Response{Header: header, Body: b}
	}
	if start >= count {
		header.Set("Content-Range", fmt.Sprintf("bytes */%d", count))
		return response.Response{Status: http.StatusRequestedRangeNotSatisfiable, Header: header}
	}

	header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, count))
	return response.Response{Status: http.StatusPartialContent, Header: header, Body: b[start : end+1]}
}

var byteRangePattern = regexp.MustCompile(`^bytes=(\d*)-(\d*)$`)

// parseByteRange parses a Range header with a single byte range, for content of the given size. It returns the
// inclusive start and end offsets, with end clamped to the content. The start can be past the content, which isn't
// satisfiable.
func parseByteRange(header string, size int) (int, int, bool) {
	m := byteRangePattern.FindStringSubmatch(strings.TrimSpace(header))
	if m == nil || (m[1] == "" && m[2] == "") {
		return 0, 0, false
	}

	if m[1] == "" {
		// A suffix range, like `bytes=-10` for the last 10 bytes.
		suffix, err := strconv.Atoi(m[2])
		if err != nil || suffix == 0 {
			return 0, 0, false
		}
		return max(size-suffix, 0), size - 1, true
	}

	start, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, false
	}
	end := size - 1
	if m[2] != "" {
		end, err = strconv.Atoi(m[2])
		if err != nil || end < start {
			return 0, 0, false
		}
		end = min(end, size-1)
	}
	return start, end, true
}

func handleInfo(ex *ex.Exchange) response.Response {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "Error: " + err.Error()
	}

	// Only env variables explicitly meant for this, so secrets in the environment aren't leaked.
	env := ex.ServerSpec.InfoEnv
	if env == nil {
		env = map[string]string{}
	}

	return response.Response{
		Body: map[string]any{
			"hostname": hostname,
			"env":      env,
		},
	}
}
