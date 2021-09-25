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

const MAX_REDIRECT_COUNT = 20

func MakeBunHandler() mux.Mux {
	m := mux.Mux{
		Storage: storage.NewMemoryStorage(),
	}

	m.HandleFunc("/", func(ex *exchange.Exchange) {
		ex.ResponseWriter.Header().Set("Content-Type", "text/html")
		assets.Render("index.html", ex.ResponseWriter, ex.Request)
	})

	m.HandleFunc("/(?P<name>.+\\.(png|ico|webmanifest))", func(ex *exchange.Exchange) {
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
	fmt.Fprintln(ex.ResponseWriter, "ok")
}

type InfoJsonOptions struct {
	Method   bool
	BodyInfo bool
}

func handleValidMethod(ex *exchange.Exchange) {
	allowedMethod := strings.TrimPrefix(ex.Request.URL.Path, "/")
	if !strings.EqualFold(ex.Request.Method, allowedMethod) {
		ex.ResponseWriter.Header().Set("Allow", allowedMethod)
		ex.ResponseWriter.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	isNonGet := ex.Request.Method != http.MethodGet
	sendInfoJson(ex, InfoJsonOptions{
		Method:   false,
		BodyInfo: isNonGet,
	})
}

func handleAnything(ex *exchange.Exchange) {
	sendInfoJson(ex, InfoJsonOptions{
		Method:   true,
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
		"args":    args,
		"headers": ex.ExposableHeadersMap(),
		"origin":  ex.FindOrigin(),
		"url":     ex.FullUrl(),
	}

	if options.Method {
		result["method"] = ex.Request.Method
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
				fmt.Println("Error reading multipart form data", err)
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
	fmt.Fprintf(ex.ResponseWriter, "%d %s", codeNum, http.StatusText(codeNum))
}

func handleAuthBasic(ex *exchange.Exchange) {
	givenUsername, givenPassword, ok := ex.Request.BasicAuth()

	if ok && givenUsername == ex.Field("user") && givenPassword == ex.Field("pass") {
		util.WriteJson(ex.ResponseWriter, map[string]interface{}{
			"authenticated": true,
			"user":          givenUsername,
		})

	} else {
		ex.ResponseWriter.Header().Set("WWW-Authenticate", "Basic realm=\"Fake Realm\"")
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)

	}
}

func handleAuthBearer(ex *exchange.Exchange) {
	expectedToken := ex.Field("tok")

	authHeader := ex.HeaderValueLast("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		ex.ResponseWriter.Header().Set("WWW-Authenticate", "Bearer")
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	util.WriteJson(ex.ResponseWriter, map[string]interface{}{
		"authenticated": token != "" && (expectedToken == "" || expectedToken == token),
		"token":         token,
	})
}

func handleAuthDigest(ex *exchange.Exchange) {
	expectedQop, expectedUsername, expectedPassword := ex.Field("qop"), ex.Field("user"), ex.Field("pass")
	newNonce := util.RandomString()
	opaque := util.RandomString()
	realm := "Digest realm=\"testrealm@host.com\", qop=\"auth,auth-int\", nonce=\"" + newNonce +
		"\", opaque=\"" + opaque + "\", algorithm=MD5, stale=FALSE"

	var authHeader string
	if vals := ex.Request.Header["Authorization"]; vals != nil && len(vals) == 1 {
		authHeader = vals[0]
	} else {
		ex.ResponseWriter.Header().Set("WWW-Authenticate", realm)
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)
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

	expectedNonce, err := ex.Request.Cookie("nonce")
	if err != nil {
		ex.ResponseWriter.Header().Set("WWW-Authenticate", realm)
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(ex.ResponseWriter, "Error: %q\n", err.Error())
		return
	}

	if givenNonce != expectedNonce.Value {
		ex.ResponseWriter.Header().Set("WWW-Authenticate", realm)
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(ex.ResponseWriter, "Error: %q\nGiven: %q\nExpected: %q", "Nonce mismatch", givenNonce, expectedNonce.Value)
		return
	}

	expectedResponseCode := computeDigestAuthResponse(
		expectedUsername,
		expectedPassword,
		expectedNonce.Value,
		givenDetails["nc"],
		givenDetails["cnonce"],
		expectedQop,
		ex.Request.Method,
		ex.Request.URL.Path,
	)

	givenResponseCode := givenDetails["response"]

	if expectedResponseCode != givenResponseCode {
		ex.ResponseWriter.Header().Set("WWW-Authenticate", realm)
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(ex.ResponseWriter, "Error: %q\nGiven: %q\nExpected: %q", "Response code mismatch", givenResponseCode, expectedResponseCode)
		return
	}

	util.WriteJson(ex.ResponseWriter, map[string]interface{}{
		"authenticated": true,
		"user":          expectedUsername,
	})
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

	fmt.Fprintln(ex.ResponseWriter, jsonContent)
}

func handleSampleXml(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("Content-Type", "application/xml")
	fmt.Fprintln(ex.ResponseWriter, `<?xml version='1.0' encoding='us-ascii'?>

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

func handleSampleRobotsTxt(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(ex.ResponseWriter, "User-agent: *\nDisallow: /deny")
}

func handleSampleRobotsDeny(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(ex.ResponseWriter, `
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

func handleSampleHtml(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(ex.ResponseWriter, `<!DOCTYPE html>
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

func handleSampleJson(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(ex.ResponseWriter, `{
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

func handleDecodeBase64(ex *exchange.Exchange) {
	encoded := ex.Field("encoded")
	if encoded == "" {
		encoded = "SFRUUEJVTiBpcyBhd2Vzb21lciE="
	}
	if decoded, err := base64.StdEncoding.DecodeString(encoded); err != nil {
		fmt.Fprint(ex.ResponseWriter, "Incorrect Base64 data try: 'SFRUUEJVTiBpcyBhd2Vzb21lciE='.")
	} else {
		fmt.Fprint(ex.ResponseWriter, string(decoded))
	}
}

func handleRandomBytes(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("content-type", "application/octet-stream")
	n, _ := strconv.Atoi(ex.Field("size"))
	ex.ResponseWriter.Write(util.RandomBytes(n))
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
		fmt.Fprintln(ex.ResponseWriter, err.Error())
		return
	}

	numbytes, err := ex.QueryParamInt("numbytes", 10)
	if err != nil {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(ex.ResponseWriter, err.Error())
		return
	}

	code, err := ex.QueryParamInt("code", 200)
	if err != nil {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(ex.ResponseWriter, err.Error())
		return
	}

	delay, err := ex.QueryParamInt("delay", 2)
	if err != nil {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(ex.ResponseWriter, err.Error())
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
		fmt.Fprint(ex.ResponseWriter, "*")
		if writeNewLines {
			fmt.Fprint(ex.ResponseWriter, "\n")
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

	fmt.Fprint(ex.ResponseWriter, "<html><head><title>Links</title></head><body>")
	for i := 0; i < count; i++ {
		if offset == i {
			fmt.Fprint(ex.ResponseWriter, i)
		} else {
			fmt.Fprintf(ex.ResponseWriter, "<a href='/links/%d/%d'>%d</a>", count, i, i)
		}
		fmt.Fprint(ex.ResponseWriter, " ")
	}
	fmt.Fprint(ex.ResponseWriter, "</body></html>")
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
		ex.ResponseWriter.Write(b)
	}
}

func handleCookies(ex *exchange.Exchange) {
	items := make(map[string]string)
	for _, cookie := range ex.Request.Cookies() {
		items[cookie.Name] = cookie.Value
	}
	util.WriteJson(ex.ResponseWriter, map[string]interface{}{
		"cookies": items,
	})
}

func handleCookiesDelete(ex *exchange.Exchange) {
	for name, _ := range ex.Request.URL.Query() {
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:    name,
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
			MaxAge:  -1, // This will produce `Max-Age: 0` in the cookie.
		})
	}

	ex.Redirect(ex.ResponseWriter, "/cookies")
}

func handleCookiesSet(ex *exchange.Exchange) {
	if ex.Field("name") == "" {
		for name, values := range ex.Request.URL.Query() {
			http.SetCookie(ex.ResponseWriter, &http.Cookie{
				Name:  name,
				Value: values[0],
				Path:  "/",
			})
		}

	} else {
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  ex.Field("name"),
			Value: ex.Field("value"),
			Path:  "/",
		})

	}

	ex.Redirect(ex.ResponseWriter, "/cookies")
}

func handleRedirectTo(ex *exchange.Exchange) {
	urls := ex.Request.URL.Query()["url"]
	if len(urls) < 1 || urls[0] == "" {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(ex.ResponseWriter, "Need url parameter")
		return
	}

	url := urls[0]
	statusCodes := ex.Request.URL.Query()["status_code"]
	statusCode := http.StatusFound
	if statusCodes != nil {
		var err error
		if statusCode, err = strconv.Atoi(statusCodes[0]); err != nil {
			ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(ex.ResponseWriter, "status_code must be an integer")
			return
		}
		if statusCode < 300 || statusCode > 399 {
			statusCode = 302
		}
	}

	ex.ResponseWriter.Header().Set("Location", url)
	ex.ResponseWriter.WriteHeader(statusCode)
}

func handleAbsoluteRedirect(ex *exchange.Exchange) {
	n, _ := strconv.Atoi(ex.Field("count"))

	if n > MAX_REDIRECT_COUNT {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(ex.ResponseWriter, "No more than %v redirects allowed.\n", MAX_REDIRECT_COUNT)
	} else if n > 1 {
		ex.Redirect(ex.ResponseWriter, regexp.MustCompile("/\\d+$").ReplaceAllLiteralString(ex.URL.String(), "/"+fmt.Sprint(n-1)))
	} else {
		ex.Redirect(ex.ResponseWriter, "/get")
	}
}

func handleRelativeRedirect(ex *exchange.Exchange) {
	n, _ := strconv.Atoi(ex.Field("count"))

	if n > MAX_REDIRECT_COUNT {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(ex.ResponseWriter, "No more than %v redirects allowed.\n", MAX_REDIRECT_COUNT)
	} else if n > 1 {
		ex.Redirect(ex.ResponseWriter, regexp.MustCompile("/\\d+$").ReplaceAllLiteralString(ex.Request.URL.Path, "/"+fmt.Sprint(n-1)))
	} else {
		ex.Redirect(ex.ResponseWriter, "/get")
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

func handleFrame(ex *exchange.Exchange) {
	embedUrl, _ := ex.QueryParamSingle("url")

	ex.ResponseWriter.Header().Set("Content-Type", "text/html")

	warning := ""
	if ex.URL.Scheme == "http" && strings.HasPrefix(embedUrl, "https://") {
		warning = `
		<p>You are embedding an https URL inside an http page, switch to full https for best experience.
		<a href='#' onclick='location.protocol = "https:"'>Click here to switch</a>.</p>`
	}

	fmt.Fprintf(ex.ResponseWriter, `<!doctype html>
<html>
<style>
html, body, form { margin: 0; min-height: 100vh }
form { display: flex; flex-direction: column }
iframe { border: none; flex-grow: 1 }
input { font-size: 1.2em; width: 100%%; margin: .5em }
p { margin: .5em }
</style>
<form>
<input name=url value='%s' placeholder='Enter URL to embed in an iframe' autofocus required>
%s
<button style='display:none'>Embed</button>
<iframe src="%s"></iframe>
</form>
`, embedUrl, warning, embedUrl)
}

func handleOauthAuthorize(ex *exchange.Exchange) {
	// Ref: <https://datatracker.ietf.org/doc/html/rfc6749>.

	// TODO: Handle POST also, where params are read from the body.
	if ex.Request.Method != http.MethodGet {
		ex.ResponseWriter.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintln(ex.ResponseWriter, http.StatusText(http.StatusMethodNotAllowed))
	}

	errors := []string{}
	params := ex.Request.URL.Query()

	redirectUrl, err := ex.QueryParamSingle("redirect_uri")
	if err != nil {
		errors = append(errors, err.Error())
	} else if !strings.HasPrefix(redirectUrl, "http://") && !strings.HasPrefix(redirectUrl, "https://") {
		errors = append(errors, "The `redirect_uri` must be an absolute URL, and should start with `http://` or `https://`.")
	}

	responseType, err := ex.QueryParamSingle("response_type")
	if err != nil {
		errors = append(errors, err.Error())
	}

	// clientId, err := ex.QueryParamSingle("client_id")
	// if err != nil {
	// 	// Required if responseType is "code" or "token"
	// 	errors = append(errors, err.Error())
	// }

	state := ""
	if len(params["state"]) > 0 {
		state = params["state"][0]
	}

	var scopes []string
	if len(params["scope"]) > 0 {
		scopes = strings.Split(strings.Join(params["scope"], " "), " ")
	}

	if len(errors) > 0 {
		ex.ResponseWriter.WriteHeader(http.StatusBadRequest)
	}

	// TODO: Error handling as per <https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2.1>.
	assets.Render("oauth-consent.html", ex.ResponseWriter, map[string]interface{}{
		"Errors":       errors,
		"scopes":       scopes,
		"redirectUrl":  redirectUrl,
		"responseType": responseType,
		"state":        state,
	})
}

func handleOauthAuthorizeSubmit(ex *exchange.Exchange) {
	if ex.Request.Method != http.MethodPost {
		ex.ResponseWriter.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintln(ex.ResponseWriter, http.StatusText(http.StatusMethodNotAllowed))
	}

	// TODO: Error out if there's *any* query params here.
	ex.Request.ParseForm()
	decision, _ := ex.FormParamSingle("decision")

	redirectUrl, _ := ex.FormParamSingle("redirect_uri")
	responseType, _ := ex.FormParamSingle("response_type")
	state, _ := ex.FormParamSingle("state")

	params := []string{}

	if state != "" {
		params = append(params, "state="+url.QueryEscape(state))
	}

	if len(ex.Request.Form["scope"]) > 0 {
		params = append(params, "scope="+url.QueryEscape(strings.Join(ex.Request.Form["scope"], " ")))
	}

	if decision == "Approve" {
		if responseType == "code" {
			params = append(params, "code=123")
		} else if responseType == "token" {
			params = append(params, "access_token=456")
			params = append(params, "token_type=bearer")
		} else {
			params = append(params, "approved=true")
		}
	} else {
		params = append(params, "error=access_denied")
	}

	ex.Redirect(ex.ResponseWriter, redirectUrl+"?"+strings.Join(params, "&"))
}

func handleInboxPush(ex *exchange.Exchange) {
	ex.Storage.PushRequestToInbox(ex.Field("name"), *ex.Request)
}

func handleInboxView(ex *exchange.Exchange) {
	name := ex.Field("name")
	entries := ex.Storage.GetFromInbox(name)
	assets.Render("inbox-view.html", ex.ResponseWriter, template.JS(util.ToJsonMust(map[string]interface{}{
		"name":    name,
		"entries": entries,
	})))
}
