package main

// Also: <https://jsonplaceholder.typicode.com/>.
// Endpoints that respond with data from SherlockHolmes or Shakespeare stories?

// A hook+inbox system, like requestbin (requestbin.net), implemented in the style of mailinator inboxes.

import (
	"bytes"
	"html/template"
	"crypto/md5"
	crypto_rand "crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sharat87/httpbun/mux"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	Version string
	Commit string
	Date string
)

//go:embed assets/*
var assets embed.FS

func main() {
	rand.Seed(time.Now().Unix())

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	if port == "" {
		port = "3090"
	}

	m := makeBunHandler()
	m.BeforeRequest = func(w http.ResponseWriter, req *mux.Request) {
		log.Printf("Handling %s %s", req.Method, req.URL.String())
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		// TODO: Include version number in the `X-Powered-By` header.
		w.Header().Set("X-Powered-By", "httpbun")
	}

	s := &http.Server{
		Addr:    host + ":" + port,
		Handler: m,
	}

	fmt.Printf("Serving on %s:%s (set HOST / PORT environment variables to change)...\n", host, port)
	fmt.Printf("Version: %q, Commit: %q, Date: %q.\n", Version, Commit, Date)
	fmt.Printf("OS: %q, Arch: %q.\n", runtime.GOOS, runtime.GOARCH)
	log.Fatal(s.ListenAndServe())
}

func makeBunHandler() mux.Mux {
	m := mux.New()

	tpl, err := template.ParseFS(assets, "assets/*.html")
	if err != nil {
		log.Fatalf("Error parsing HTML assets %v.", err)
	}

	var indexHtmlBytes bytes.Buffer
	if err := tpl.ExecuteTemplate(&indexHtmlBytes, "index.html", nil); err != nil {
		log.Fatalf("Error executing index.html template %v.", err)
	}
	indexHtml := indexHtmlBytes.String()

	m.HandleFunc("/", func(w http.ResponseWriter, req *mux.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, indexHtml)
	})

	m.HandleFunc("/(?P<name>.+\\.(?P<ext>png|ico|webmanifest))", func(w http.ResponseWriter, req *mux.Request) {
		name := req.Field("name")
		// ext := req.Field("ext")
		if content, err := assets.ReadFile("assets/" + name); err == nil {
			w.Write(content)
		} else if strings.HasSuffix(err.Error(), " file does not exist") {
			http.NotFound(w, &req.Request)
		} else {
			log.Printf("Error %v", err)
		}
	})

	m.HandleFunc("/get", handleValidMethod)
	m.HandleFunc("/post", handleValidMethod)
	m.HandleFunc("/put", handleValidMethod)
	m.HandleFunc("/patch", handleValidMethod)
	m.HandleFunc("/delete", handleValidMethod)

	m.HandleFunc("/headers", handleHeaders)

	m.HandleFunc("/basic-auth/(?P<user>[^/]+)/(?P<pass>[^/]+)/?", handleAuthBasic)
	m.HandleFunc("/bearer", handleAuthBearer)
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

	m.HandleFunc("/base64(/(?P<encoded>.*))?", handleDecodeBase64)
	m.HandleFunc("/bytes/(?P<size>\\d+)", handleRandomBytes)
	m.HandleFunc("/delay/(?P<delay>\\d+)", handleDelayedResponse)
	m.HandleFunc("/drip(-(?P<mode>lines))?", handleDrip)
	m.HandleFunc("/links/(?P<count>\\d+)(/(?P<offset>\\d+))?/?", handleLinks)
	m.HandleFunc("/range/(?P<count>\\d+)/?", handleRange)

	m.HandleFunc("/cookies", handleCookies)
	m.HandleFunc("/cookies/delete", handleCookiesDelete)
	m.HandleFunc("/cookies/set(/(?P<name>[^/]+)/(?P<value>[^/]+))?", handleCookiesSet)

	m.HandleFunc("/redirect-to", handleRedirectTo)
	m.HandleFunc("/(relative-)?redirect/(?P<count>\\d+)", handleRelativeRedirect)
	m.HandleFunc("/absolute-redirect/(?P<count>\\d+)", handleAbsoluteRedirect)

	m.HandleFunc("/anything\\b.*", handleAnything)

	return m
}

type InfoJsonOptions struct {
	Method bool
	Form   bool
	Data   bool
}

func handleValidMethod(w http.ResponseWriter, req *mux.Request) {
	allowedMethod := strings.TrimPrefix(req.URL.Path, "/")
	if !strings.EqualFold(req.Method, allowedMethod) {
		w.Header().Set("Allow", allowedMethod)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	isNonGet := req.Method != http.MethodGet
	sendInfoJson(w, req, InfoJsonOptions{
		Method: false,
		Form:   isNonGet,
		Data:   isNonGet,
	})
}

func handleAnything(w http.ResponseWriter, req *mux.Request) {
	sendInfoJson(w, req, InfoJsonOptions{
		Method: true,
		Form:   true,
		Data:   true,
	})
}

func handleHeaders(w http.ResponseWriter, req *mux.Request) {
	headers := make(map[string]string)
	for name, values := range req.Header {
		headers[name] = strings.Join(values, ", ")
	}

	writeJson(w, map[string]interface{}{
		"headers": headers,
	})
}

func sendInfoJson(w http.ResponseWriter, req *mux.Request, options InfoJsonOptions) {
	args := make(map[string]interface{})
	for name, values := range req.URL.Query() {
		if len(values) > 1 {
			args[name] = values
		} else {
			args[name] = values[0]
		}
	}

	headers := make(map[string]string)
	for name, values := range req.Header {
		headers[name] = strings.Join(values, ", ")
	}

	body := ""
	if bodyBytes, err := ioutil.ReadAll(req.CappedBody); err != nil {
		fmt.Println("Error reading request payload", err)
		return
	} else {
		body = string(bodyBytes)
	}

	contentType := headerValue(req, "Content-Type")

	form := make(map[string]interface{})
	data := ""

	if contentType == "application/x-www-form-urlencoded" {
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

	} else {
		data = body

	}

	result := map[string]interface{}{
		"args":    args,
		"headers": headers,
		"origin":  req.Host,
		"url":     req.URL.String(),
	}

	if options.Method {
		result["method"] = req.Method
	}

	if options.Form {
		result["form"] = form
	}

	if options.Data {
		result["data"] = data
	}

	writeJson(w, result)
}

func handleStatus(w http.ResponseWriter, req *mux.Request) {
	codes := regexp.MustCompile("\\d+").FindAllString(req.URL.String(), -1)

	var code string
	if len(codes) > 1 {
		code = codes[rand.Intn(len(codes))]
	} else {
		code = codes[0]
	}

	codeNum, _ := strconv.Atoi(code)
	w.WriteHeader(codeNum)
	fmt.Fprintf(w, "%d %s", codeNum, http.StatusText(codeNum))
}

func handleAuthBasic(w http.ResponseWriter, req *mux.Request) {
	givenUsername, givenPassword, ok := req.BasicAuth()

	if ok && givenUsername == req.Field("user") && givenPassword == req.Field("pass") {
		writeJson(w, map[string]interface{}{
			"authenticated": true,
			"user":          givenUsername,
		})

	} else {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Fake Realm\"")
		w.WriteHeader(http.StatusUnauthorized)

	}
}

func handleAuthBearer(w http.ResponseWriter, req *mux.Request) {
	authHeader := headerValue(req, "Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.Header().Set("WWW-Authenticate", "Bearer")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	writeJson(w, map[string]interface{}{
		"authenticated": true,
		"token":         token,
	})
}

func handleAuthDigest(w http.ResponseWriter, req *mux.Request) {
	expectedQop, expectedUsername, expectedPassword := req.Field("qop"), req.Field("user"), req.Field("pass")
	newNonce := randomString()
	opaque := randomString()
	realm := "Digest realm=\"testrealm@host.com\", qop=\"auth,auth-int\", nonce=\"" + newNonce +
		"\", opaque=\"" + opaque + "\", algorithm=MD5, stale=FALSE"

	var authHeader string
	if vals := req.Header["Authorization"]; vals != nil && len(vals) == 1 {
		authHeader = vals[0]
	} else {
		w.Header().Set("WWW-Authenticate", realm)
		http.SetCookie(w, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	matches := regexp.MustCompile("([a-z]+)=(?:\"([^\"]+)\"|([^,]+))").FindAllStringSubmatch(authHeader, -1)
	givenDetails := make(map[string]string)
	for _, m := range matches {
		key := m[1]
		val := m[2]
		if val == "" {
			val = m[3]
		}
		givenDetails[key] = val
	}

	givenNonce := givenDetails["nonce"]

	expectedNonce, err := req.Cookie("nonce")
	if err != nil {
		w.Header().Set("WWW-Authenticate", realm)
		http.SetCookie(w, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Error: %q\n", err.Error())
		return
	}

	if givenNonce != expectedNonce.Value {
		w.Header().Set("WWW-Authenticate", realm)
		http.SetCookie(w, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Error: %q\nGiven: %q\nExpected: %q", "Nonce mismatch", givenNonce, expectedNonce.Value)
		return
	}

	expectedResponseCode := computeDigestAuthResponse(
		expectedUsername,
		expectedPassword,
		expectedNonce.Value,
		givenDetails["nc"],
		givenDetails["cnonce"],
		expectedQop,
		req.Method,
		req.URL.Path,
	)

	givenResponseCode := givenDetails["response"]

	if expectedResponseCode != givenResponseCode {
		w.Header().Set("WWW-Authenticate", realm)
		http.SetCookie(w, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Error: %q\nGiven: %q\nExpected: %q", "Response code mismatch", givenResponseCode, expectedResponseCode)
		return
	}

	writeJson(w, map[string]interface{}{
		"authenticated": true,
		"user":          expectedUsername,
	})
}

// Digest auth response computer.
func computeDigestAuthResponse(username, password, serverNonce, nc, clientNonce, qop, method, path string) string {
	// Source: <https://en.wikipedia.org/wiki/Digest_access_authentication>.
	ha1 := md5sum(username + ":" + "testrealm@host.com" + ":" + password)
	ha2 := md5sum(method + ":" + path)
	return md5sum(ha1 + ":" + serverNonce + ":" + nc + ":" + clientNonce + ":" + qop + ":" + ha2)
}

func handleIp(w http.ResponseWriter, req *mux.Request) {
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.Printf("Unable to read IP from address %q.", req.RemoteAddr)
		return
	}

	userIP := net.ParseIP(ip)
	ipStr := ""
	if userIP != nil {
		ipStr = fmt.Sprint(w, userIP)
	}

	writeJson(w, map[string]string{
		"origin": ipStr,
	})
}

func handleUserAgent(w http.ResponseWriter, req *mux.Request) {
	writeJson(w, map[string]string{
		"user-agent": headerValue(req, "User-Agent"),
	})
}

func handleCache(w http.ResponseWriter, req *mux.Request) {
	shouldSendData :=
		headerValue(req, "If-Modified-Since") == "" &&
			headerValue(req, "If-None-Match") == ""

	if shouldSendData {
		isNonGet := req.Method != http.MethodGet
		sendInfoJson(w, req, InfoJsonOptions{
			Form: isNonGet,
			Data: isNonGet,
		})
	} else {
		w.WriteHeader(http.StatusNotModified)
	}
}

func handleCacheControl(w http.ResponseWriter, req *mux.Request) {
	w.Header().Set("Cache-Control", "public, max-age="+req.Field("age"))
	isNonGet := req.Method != http.MethodGet
	sendInfoJson(w, req, InfoJsonOptions{
		Form: isNonGet,
		Data: isNonGet,
	})
}

func handleEtag(w http.ResponseWriter, req *mux.Request) {
	// TODO: Handle If-Match header in etag endpoint: <https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Match>.
	etagInUrl := req.Field("etag")
	etagInHeader := headerValue(req, "If-None-Match")

	if etagInUrl == etagInHeader {
		w.WriteHeader(http.StatusNotModified)
	} else {
		isNonGet := req.Method != http.MethodGet
		sendInfoJson(w, req, InfoJsonOptions{
			Form: isNonGet,
			Data: isNonGet,
		})
	}
}

func handleResponseHeaders(w http.ResponseWriter, req *mux.Request) {
	data := make(map[string]interface{})

	for name, values := range req.URL.Query() {
		name = http.CanonicalHeaderKey(name)
		for _, value := range values {
			w.Header().Add(name, value)
		}
		if len(values) > 1 {
			data[name] = values
		} else {
			data[name] = values[0]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	data["Content-Type"] = "application/json"

	jsonContent := ""

	for {
		jsonContent = toJsonMust(data)
		newContentLength := fmt.Sprint(len(jsonContent))
		if data["Content-Length"] == newContentLength {
			break
		}
		data["Content-Length"] = newContentLength
	}

	fmt.Fprintln(w, jsonContent)
}

func handleSampleXml(w http.ResponseWriter, req *mux.Request) {
	w.Header().Set("Content-Type", "application/xml")
	fmt.Fprintln(w, `<?xml version='1.0' encoding='us-ascii'?>

<!--  A SAMPLE set of slides  -->

<slideshow 
    title="Sample Slide Show"
    date="Date of publication"
    author="Yours Truly"
    >

    <!-- TITLE SLIDE -->
    <slide type="all">
      <title>Wake up to WonderWidgets!</title>
    </slide>

    <!-- OVERVIEW -->
    <slide type="all">
        <title>Overview</title>
        <item>Why <em>WonderWidgets</em> are great</item>
        <item/>
        <item>Who <em>buys</em> WonderWidgets</item>
    </slide>

</slideshow>`)
}

func handleSampleRobotsTxt(w http.ResponseWriter, req *mux.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "User-agent: *\nDisallow: /deny")
}

func handleSampleRobotsDeny(w http.ResponseWriter, req *mux.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, `
          .-''''''-.
        .' _      _ '.
       /   O      O   \
      :                :
      |                |
      :       __       :
       \  .-"`+"`  `"+`"-.  /
        '.          .'
          '-......-'
     YOU SHOULDN'T BE HERE`)
}

func handleSampleHtml(w http.ResponseWriter, req *mux.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, `<!DOCTYPE html>
<html>
  <head>
  </head>
  <body>
      <h1>Herman Melville - Moby-Dick</h1>

      <div>
        <p>
          Availing himself of the mild, summer-cool weather that now reigned in these latitudes, and in preparation for the peculiarly active pursuits shortly to be anticipated, Perth, the begrimed, blistered old blacksmith, had not removed his portable forge to the hold again, after concluding his contributory work for Ahab's leg, but still retained it on deck, fast lashed to ringbolts by the foremast; being now almost incessantly invoked by the headsmen, and harpooneers, and bowsmen to do some little job for them; altering, or repairing, or new shaping their various weapons and boat furniture. Often he would be surrounded by an eager circle, all waiting to be served; holding boat-spades, pike-heads, harpoons, and lances, and jealously watching his every sooty movement, as he toiled. Nevertheless, this old man's was a patient hammer wielded by a patient arm. No murmur, no impatience, no petulance did come from him. Silent, slow, and solemn; bowing over still further his chronically broken back, he toiled away, as if toil were life itself, and the heavy beating of his hammer the heavy beating of his heart. And so it was.—Most miserable! A peculiar walk in this old man, a certain slight but painful appearing yawing in his gait, had at an early period of the voyage excited the curiosity of the mariners. And to the importunity of their persisted questionings he had finally given in; and so it came to pass that every one now knew the shameful story of his wretched fate. Belated, and not innocently, one bitter winter's midnight, on the road running between two country towns, the blacksmith half-stupidly felt the deadly numbness stealing over him, and sought refuge in a leaning, dilapidated barn. The issue was, the loss of the extremities of both feet. Out of this revelation, part by part, at last came out the four acts of the gladness, and the one long, and as yet uncatastrophied fifth act of the grief of his life's drama. He was an old man, who, at the age of nearly sixty, had postponedly encountered that thing in sorrow's technicals called ruin. He had been an artisan of famed excellence, and with plenty to do; owned a house and garden; embraced a youthful, daughter-like, loving wife, and three blithe, ruddy children; every Sunday went to a cheerful-looking church, planted in a grove. But one night, under cover of darkness, and further concealed in a most cunning disguisement, a desperate burglar slid into his happy home, and robbed them all of everything. And darker yet to tell, the blacksmith himself did ignorantly conduct this burglar into his family's heart. It was the Bottle Conjuror! Upon the opening of that fatal cork, forth flew the fiend, and shrivelled up his home. Now, for prudent, most wise, and economic reasons, the blacksmith's shop was in the basement of his dwelling, but with a separate entrance to it; so that always had the young and loving healthy wife listened with no unhappy nervousness, but with vigorous pleasure, to the stout ringing of her young-armed old husband's hammer; whose reverberations, muffled by passing through the floors and walls, came up to her, not unsweetly, in her nursery; and so, to stout Labor's iron lullaby, the blacksmith's infants were rocked to slumber. Oh, woe on woe! Oh, Death, why canst thou not sometimes be timely? Hadst thou taken this old blacksmith to thyself ere his full ruin came upon him, then had the young widow had a delicious grief, and her orphans a truly venerable, legendary sire to dream of in their after years; and all of them a care-killing competency.
        </p>
      </div>
  </body>
</html>`)
}

func handleSampleJson(w http.ResponseWriter, req *mux.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{
  "slideshow": {
    "author": "Yours Truly",
    "date": "date of publication",
    "slides": [
      {
        "title": "Wake up to WonderWidgets!",
        "type": "all"
      },
      {
        "items": [
          "Why <em>WonderWidgets</em> are great",
          "Who <em>buys</em> WonderWidgets"
        ],
        "title": "Overview",
        "type": "all"
      }
    ],
    "title": "Sample Slide Show"
  }
}`)
}

func handleDecodeBase64(w http.ResponseWriter, req *mux.Request) {
	encoded := req.Field("encoded")
	if encoded == "" {
		encoded = "SFRUUEJVTiBpcyBhd2Vzb21lciE="
	}
	if decoded, err := base64.StdEncoding.DecodeString(encoded); err != nil {
		fmt.Fprint(w, "Incorrect Base64 data try: 'SFRUUEJVTiBpcyBhd2Vzb21lciE='.")
	} else {
		fmt.Fprint(w, string(decoded))
	}
}

func handleRandomBytes(w http.ResponseWriter, req *mux.Request) {
	w.Header().Set("content-type", "application/octet-stream")
	n, _ := strconv.Atoi(req.Field("size"))
	w.Write(randomBytes(n))
}

func handleDelayedResponse(w http.ResponseWriter, req *mux.Request) {
	n, _ := strconv.Atoi(req.Field("delay"))
	time.Sleep(time.Duration(n) * time.Second)
}

func handleDrip(w http.ResponseWriter, req *mux.Request) {
	// Test with `curl -N localhost:3090/drip`.

	writeNewLines := req.Field("mode") == "lines"

	duration, err := queryParamInt(req, "duration", 2)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err.Error())
		return
	}

	numbytes, err := queryParamInt(req, "numbytes", 10)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err.Error())
		return
	}

	code, err := queryParamInt(req, "code", 200)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err.Error())
		return
	}

	delay, err := queryParamInt(req, "delay", 2)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err.Error())
		return
	}

	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Second)
	}
	w.WriteHeader(code)

	interval := time.Duration(float32(time.Second) * float32(duration) / float32(numbytes))

	for numbytes > 0 {
		fmt.Fprint(w, "*")
		if writeNewLines {
			fmt.Fprint(w, "\n")
		}
		if !flush(w) {
			log.Println("Flush not available. Dripping and streaming not supported on this platform.")
		}
		time.Sleep(interval)
		numbytes--
	}
}

func handleLinks(w http.ResponseWriter, req *mux.Request) {
	count, _ := strconv.Atoi(req.Field("count"))
	offset, _ := strconv.Atoi(req.Field("offset"))

	fmt.Fprint(w, "<html><head><title>Links</title></head><body>")
	for i := 0; i < count; i++ {
		if offset == i {
			fmt.Fprint(w, i)
		} else {
			fmt.Fprintf(w, "<a href='/links/%d/%d'>%d</a>", count, i, i)
		}
		fmt.Fprint(w, " ")
	}
	fmt.Fprint(w, "</body></html>")
}

func handleRange(w http.ResponseWriter, req *mux.Request) {
	// TODO: Cache range response, don't have to generate over and over again.
	count, _ := strconv.Atoi(req.Field("count"))

	if count > 1000 {
		count = 1000
	} else if count < 0 {
		count = 0
	}

	w.Header().Set("content-type", "application/octet-stream")

	if count > 0 {
		r := rand.New(rand.NewSource(42))
		b := make([]byte, count)
		r.Read(b)
		w.Write(b)
	}
}

func handleCookies(w http.ResponseWriter, req *mux.Request) {
	items := make(map[string]string)
	for _, cookie := range req.Cookies() {
		items[cookie.Name] = cookie.Value
	}
	writeJson(w, map[string]interface{}{
		"cookies": items,
	})
}

func handleCookiesDelete(w http.ResponseWriter, req *mux.Request) {
	for name, _ := range req.URL.Query() {
		http.SetCookie(w, &http.Cookie{
			Name:    name,
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
			MaxAge:  -1, // This will produce `Max-Age: 0` in the cookie.
		})
	}

	redirect(w, req, "/cookies")
}

func handleCookiesSet(w http.ResponseWriter, req *mux.Request) {
	if req.Field("name") == "" {
		for name, values := range req.URL.Query() {
			http.SetCookie(w, &http.Cookie{
				Name:  name,
				Value: values[0],
				Path:  "/",
			})
		}

	} else {
		http.SetCookie(w, &http.Cookie{
			Name:  req.Field("name"),
			Value: req.Field("value"),
			Path:  "/",
		})

	}

	redirect(w, req, "/cookies")
}

func handleRedirectTo(w http.ResponseWriter, req *mux.Request) {
	urls := req.URL.Query()["url"]
	if len(urls) < 1 || urls[0] == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Need url parameter")
		return
	}

	url := urls[0]
	statusCodes := req.URL.Query()["status_code"]
	statusCode := http.StatusFound
	if statusCodes != nil {
		var err error
		if statusCode, err = strconv.Atoi(statusCodes[0]); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "status_code must be an integer")
			return
		}
		if statusCode < 300 || statusCode > 399 {
			statusCode = 302
		}
	}

	w.Header().Set("Location", url)
	w.WriteHeader(statusCode)
}

func handleAbsoluteRedirect(w http.ResponseWriter, req *mux.Request) {
	n, _ := strconv.Atoi(req.Field("count"))

	if n > 1 {
		redirect(w, req, regexp.MustCompile("/\\d+$").ReplaceAllLiteralString(req.URL.String(), "/"+fmt.Sprint(n-1)))
	} else {
		redirect(w, req, "/get")
	}
}

func handleRelativeRedirect(w http.ResponseWriter, req *mux.Request) {
	n, _ := strconv.Atoi(req.Field("count"))

	if n > 1 {
		redirect(w, req, regexp.MustCompile("/\\d+$").ReplaceAllLiteralString(req.URL.Path, "/"+fmt.Sprint(n-1)))
	} else {
		redirect(w, req, "/get")
	}
}

func redirect(w http.ResponseWriter, req *mux.Request, path string) {
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

func headerValue(req *mux.Request, name string) string {
	if req == nil {
		return ""
	}

	if values := req.Header[name]; values != nil && len(values) > 0 {
		return values[len(values)-1]
	}

	return ""
}

func queryParamInt(req *mux.Request, name string, value int) (int, error) {
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

func writeJson(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, toJsonMust(data))
}

func toJsonMust(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}

func randomBytes(n int) []byte {
	b := make([]byte, n)

	if _, err := crypto_rand.Read(b); err != nil {
		fmt.Println("Error: ", err)
		return []byte{}
	}

	return b[:]
}

func randomString() string {
	return hex.EncodeToString(randomBytes(16))
}

func md5sum(text string) string {
	// Source: <https://stackoverflow.com/a/25286918/151048>.
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func flush(w http.ResponseWriter) bool {
	f, ok := w.(http.Flusher)
	if ok {
		f.Flush()
	}
	return ok
}
