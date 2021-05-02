package main

// Also: <https://jsonplaceholder.typicode.com/>.
// Endpoints that respond with data from SherlockHolmes or Shakespeare stories?

import (
	"crypto/md5"
	crypto_rand "crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sharat87/httpbun/mux"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed static/*
var statics embed.FS

func main() {
	rand.Seed(time.Now().Unix())

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	if port == "" {
		port = "3090"
	}

	s := &http.Server{
		Addr:    host + ":" + port,
		Handler: makeBunHandler(),
	}

	fmt.Printf("Serving on %s:%s (set HOST / PORT environment variables to change)...\n", host, port)
	log.Fatal(s.ListenAndServe())
}

func makeBunHandler() http.Handler {
	mux := mux.New()

	tpl, err := template.ParseFS(statics, "static/*.html")
	if err != nil {
		panic(err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request, params map[string]string) {
		w.Header().Set("Content-Type", "text/html")
		tpl.ExecuteTemplate(w, "index.html", mux)
	}, "")

	mux.HandleFunc("/get", handleValidMethod, `
	Accepts GET requests and responds with a JSON object with query params, headers and a few other information about
	the request.
	`)
	mux.HandleFunc("/head", handleValidMethod, `
	Accepts POST requests and responds with a JSON object with form body, query params, headers and a few other
	information about the request.
	`)
	mux.HandleFunc("/post", handleValidMethod, "")
	mux.HandleFunc("/put", handleValidMethod, "")
	mux.HandleFunc("/patch", handleValidMethod, "")
	mux.HandleFunc("/delete", handleValidMethod, "")

	mux.HandleFunc("/basic-auth/(?P<user>[^/]+)/(?P<pass>[^/]+)/?", handleAuthBasic, "")
	mux.HandleFunc("/bearer", handleAuthBearer, "")
	mux.HandleFunc("/digest-auth/(?P<qop>[^/]+)/(?P<user>[^/]+)/(?P<pass>[^/]+)/?", handleAuthDigest, "")

	mux.HandleFunc("/status/[\\d,]+", handleStatus, "")
	mux.HandleFunc("/ip", handleIp, "")
	mux.HandleFunc("/user-agent", handleUserAgent, "")

	mux.HandleFunc("/cache", handleCache, "")
	mux.HandleFunc("/cache/(?P<age>\\d+)", handleCacheControl, "")
	mux.HandleFunc("/etag/(?P<etag>[^/]+)", handleEtag, "")
	mux.HandleFunc("/response-headers", handleResponseHeaders, "")

	mux.HandleFunc("/deny", handleSampleRobotsDeny, "")
	mux.HandleFunc("/html", handleSampleHtml, "")
	mux.HandleFunc("/json", handleSampleJson, "")
	mux.HandleFunc("/robots.txt", handleSampleRobotsTxt, "")
	mux.HandleFunc("/xml", handleSampleXml, "")

	mux.HandleFunc("/base64(/(?P<encoded>.*))?", handleDecodeBase64, "")
	mux.HandleFunc("/bytes/(?P<size>\\d+)", handleRandomBytes, "")
	mux.HandleFunc("/delay/(?P<delay>\\d+)", handleDelayedResponse, "")
	mux.HandleFunc("/drip", handleDrip, "")
	mux.HandleFunc("/links/(?P<count>\\d+)(/(?P<offset>\\d+))?/?", handleLinks, "")
	mux.HandleFunc("/range/(?P<count>\\d+)/?", handleRange, "")

	mux.HandleFunc("/cookies", handleCookies, "")
	mux.HandleFunc("/cookies/delete", handleCookiesDelete, "")
	mux.HandleFunc("/cookies/set(/(?P<name>[^/]+)/(?P<value>[^/]+))?", handleCookiesSet, "")

	mux.HandleFunc("/redirect-to", handleRedirectTo, "")
	mux.HandleFunc("/(relative-)?redirect/(?P<count>\\d+)", handleRelativeRedirect, "")
	mux.HandleFunc("/absolute-redirect/(?P<count>\\d+)", handleAbsoluteRedirect, "")

	mux.HandleFunc("/anything\\b.*", handleAnything, "")

	return mux
}

type InfoJsonOptions struct {
	Method bool
	Form   bool
	Data   bool
}

func handleValidMethod(w http.ResponseWriter, req *http.Request, params map[string]string) {
	if !strings.EqualFold(req.Method, strings.TrimPrefix(req.URL.Path, "/")) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	isNonGet := req.Method != http.MethodGet
	sendInfoJson(w, req, InfoJsonOptions{
		Method: false,
		Form:   isNonGet,
		Data:   isNonGet,
	})
}

func handleAnything(w http.ResponseWriter, req *http.Request, params map[string]string) {
	sendInfoJson(w, req, InfoJsonOptions{
		Method: true,
		Form:   true,
		Data:   true,
	})
}

func sendInfoJson(w http.ResponseWriter, req *http.Request, options InfoJsonOptions) {
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
	if bodyBytes, err := ioutil.ReadAll(io.LimitReader(req.Body, 10000)); err != nil {
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

func handleStatus(w http.ResponseWriter, req *http.Request, params map[string]string) {
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

func handleAuthBasic(w http.ResponseWriter, req *http.Request, params map[string]string) {
	givenUsername, givenPassword, ok := req.BasicAuth()

	if ok && givenUsername == params["user"] && givenPassword == params["pass"] {
		writeJson(w, map[string]interface{}{
			"authenticated": true,
			"user":          givenUsername,
		})

	} else {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Fake Realm\"")
		w.WriteHeader(http.StatusUnauthorized)

	}
}

func handleAuthBearer(w http.ResponseWriter, req *http.Request, params map[string]string) {
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

func handleAuthDigest(w http.ResponseWriter, req *http.Request, params map[string]string) {
	expectedQop, expectedUsername, expectedPassword := params["qop"], params["user"], params["pass"]
	fmt.Println("expected", expectedQop, expectedUsername, expectedPassword)

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

	fmt.Println("Authorization header", authHeader)

	matches := regexp.MustCompile("([a-z]+)=(?:\"([^\"]+)\"|([^,]+))").FindAllStringSubmatch(authHeader, -1)
	fmt.Println("Auth header match", matches)
	givenDetails := make(map[string]string)
	for _, m := range matches {
		key := m[1]
		val := m[2]
		if val == "" {
			val = m[3]
		}
		givenDetails[key] = val
	}
	fmt.Println("givenDetails", givenDetails)

	givenNonce := givenDetails["nonce"]
	fmt.Println("givenNonce", givenNonce)

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

	fmt.Println("Expected nonce", expectedNonce.Value)

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
	fmt.Println("expected ha3", expectedResponseCode)

	givenResponseCode := givenDetails["response"]
	fmt.Println("given ha3", givenResponseCode)

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

func handleIp(w http.ResponseWriter, req *http.Request, params map[string]string) {
	writeJson(w, map[string]string{
		"origin": strings.Split(req.RemoteAddr, ":")[0],
	})
}

func handleUserAgent(w http.ResponseWriter, req *http.Request, params map[string]string) {
	writeJson(w, map[string]string{
		"user-agent": headerValue(req, "User-Agent"),
	})
}

func handleCache(w http.ResponseWriter, req *http.Request, params map[string]string) {
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

func handleCacheControl(w http.ResponseWriter, req *http.Request, params map[string]string) {
	w.Header().Set("Cache-Control", "public, max-age="+params["age"])
	isNonGet := req.Method != http.MethodGet
	sendInfoJson(w, req, InfoJsonOptions{
		Form: isNonGet,
		Data: isNonGet,
	})
}

func handleEtag(w http.ResponseWriter, req *http.Request, params map[string]string) {
	etagInUrl := params["etag"]
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

func handleResponseHeaders(w http.ResponseWriter, req *http.Request, params map[string]string) {
	for name, values := range req.URL.Query() {
		w.Header().Set(name, values[0])
	}
	// TODO: JSON Body for /response-headers.
}

func handleSampleXml(w http.ResponseWriter, req *http.Request, params map[string]string) {
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

func handleSampleRobotsTxt(w http.ResponseWriter, req *http.Request, params map[string]string) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "User-agent: *\nDisallow: /deny")
}

func handleSampleRobotsDeny(w http.ResponseWriter, req *http.Request, params map[string]string) {
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

func handleSampleHtml(w http.ResponseWriter, req *http.Request, params map[string]string) {
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

func handleSampleJson(w http.ResponseWriter, req *http.Request, params map[string]string) {
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

func handleDecodeBase64(w http.ResponseWriter, req *http.Request, params map[string]string) {
	encoded := params["encoded"]
	if encoded == "" {
		encoded = "SFRUUEJVTiBpcyBhd2Vzb21lciE="
	}
	if decoded, err := base64.StdEncoding.DecodeString(encoded); err != nil {
		fmt.Fprint(w, "Incorrect Base64 data try: 'SFRUUEJVTiBpcyBhd2Vzb21lciE='.")
	} else {
		fmt.Fprint(w, string(decoded))
	}
}

func handleRandomBytes(w http.ResponseWriter, req *http.Request, params map[string]string) {
	w.Header().Set("content-type", "application/octet-stream")
	n, _ := strconv.Atoi(params["size"])
	w.Write(randomBytes(n))
}

func handleDelayedResponse(w http.ResponseWriter, req *http.Request, params map[string]string) {
	n, _ := strconv.Atoi(params["delay"])
	time.Sleep(time.Duration(n) * time.Second)
}

func handleDrip(w http.ResponseWriter, req *http.Request, params map[string]string) {
	args := req.URL.Query()
	var err error

	duration := 2
	if len(args["duration"]) > 0 {
		duration, err = strconv.Atoi(args["duration"][0])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "duration must be an integer")
			return
		}
	}

	numbytes := 10
	if len(args["numbytes"]) > 0 {
		numbytes, err = strconv.Atoi(args["numbytes"][0])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "numbytes must be an integer")
			return
		}
	}

	code := 200
	if len(args["code"]) > 0 {
		code, err = strconv.Atoi(args["code"][0])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "code must be an integer")
			return
		}
	}

	delay := 2
	if len(args["delay"]) > 0 {
		delay, err = strconv.Atoi(args["delay"][0])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "delay must be an integer")
			return
		}
	}

	time.Sleep(time.Duration(delay) * time.Second)
	w.WriteHeader(code)

	interval := time.Duration(float32(time.Second) * float32(duration) / float32(numbytes))

	for numbytes > 0 {
		fmt.Fprint(w, "*")
		time.Sleep(interval)
		numbytes--
	}
}

func handleLinks(w http.ResponseWriter, req *http.Request, params map[string]string) {
	count, _ := strconv.Atoi(params["count"])
	offset, _ := strconv.Atoi(params["offset"])

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

func handleRange(w http.ResponseWriter, req *http.Request, params map[string]string) {
	// TODO: Cache range response, don't have to generate over and over again.
	count, _ := strconv.Atoi(params["count"])
	r := rand.New(rand.NewSource(42))
	b := make([]byte, count)
	r.Read(b)

	w.Header().Set("content-type", "application/octet-stream")
	w.Write(b)
}

func handleCookies(w http.ResponseWriter, req *http.Request, params map[string]string) {
	items := make(map[string]string)
	for _, cookie := range req.Cookies() {
		items[cookie.Name] = cookie.Value
	}
	writeJson(w, map[string]interface{}{
		"cookies": items,
	})
}

func handleCookiesDelete(w http.ResponseWriter, req *http.Request, params map[string]string) {
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

func handleCookiesSet(w http.ResponseWriter, req *http.Request, params map[string]string) {
	if params["name"] == "" {
		for name, values := range req.URL.Query() {
			http.SetCookie(w, &http.Cookie{
				Name:  name,
				Value: values[0],
				Path:  "/",
			})
		}

	} else {
		http.SetCookie(w, &http.Cookie{
			Name:  params["name"],
			Value: params["value"],
			Path:  "/",
		})

	}

	redirect(w, req, "/cookies")
}

func handleRedirectTo(w http.ResponseWriter, req *http.Request, params map[string]string) {
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

func handleAbsoluteRedirect(w http.ResponseWriter, req *http.Request, params map[string]string) {
	n, _ := strconv.Atoi(params["count"])

	if n > 1 {
		redirect(w, req, regexp.MustCompile("/\\d+$").ReplaceAllLiteralString(req.URL.String(), "/"+fmt.Sprint(n-1)))
	} else {
		redirect(w, req, "/get")
	}
}

func handleRelativeRedirect(w http.ResponseWriter, req *http.Request, params map[string]string) {
	n, _ := strconv.Atoi(params["count"])

	if n > 1 {
		redirect(w, req, regexp.MustCompile("/\\d+$").ReplaceAllLiteralString(req.URL.Path, "/"+fmt.Sprint(n-1)))
	} else {
		redirect(w, req, "/get")
	}
}

func redirect(w http.ResponseWriter, req *http.Request, path string) {
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

func headerValue(req *http.Request, name string) string {
	if req == nil {
		return ""
	}

	if values := req.Header[name]; values != nil && len(values) > 0 {
		return values[len(values)-1]
	}

	return ""
}

func writeJson(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if b, err := json.Marshal(data); err != nil {
		log.Fatal(err)
	} else {
		fmt.Fprintln(w, string(b))
	}
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
