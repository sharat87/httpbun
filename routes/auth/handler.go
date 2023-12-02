package auth

import (
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/util"
	"net/http"
	"regexp"
	"strings"
)

var Routes = map[string]exchange.HandlerFn{
	"/basic-auth/(?P<user>[^/]+)/(?P<pass>[^/]+)/?":                 handleAuthBasic,
	"/bearer(/(?P<tok>\\w+))?":                                      handleAuthBearer,
	"/digest-auth/(?P<qop>[^/]+)/(?P<user>[^/]+)/(?P<pass>[^/]+)/?": handleAuthDigest,
	"/digest-auth/(?P<user>[^/]+)/(?P<pass>[^/]+)/?":                handleAuthDigest,
}

func handleAuthBasic(ex *exchange.Exchange) {
	givenUsername, givenPassword, ok := ex.Request.BasicAuth()
	isAuthenticated := ok && givenUsername == ex.Field("user") && givenPassword == ex.Field("pass")

	if !isAuthenticated {
		ex.ResponseWriter.Header().Set(c.WWWAuthenticate, "Basic realm=\"Fake Realm\"")
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		return
	}

	util.WriteJson(ex.ResponseWriter, map[string]any{
		"authenticated": isAuthenticated,
		"user":          givenUsername,
	})
}

func handleAuthBearer(ex *exchange.Exchange) {
	expectedToken := ex.Field("tok")

	authHeader := ex.HeaderValueLast("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		ex.ResponseWriter.Header().Set(c.WWWAuthenticate, "Bearer")
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	util.WriteJson(ex.ResponseWriter, map[string]any{
		"authenticated": token != "" && (expectedToken == "" || expectedToken == token),
		"token":         token,
	})
}

func handleAuthDigest(ex *exchange.Exchange) {
	expectedQop, expectedUsername, expectedPassword := ex.Field("qop"), ex.Field("user"), ex.Field("pass")

	if expectedQop == "" {
		expectedQop = "auth"
	}

	newNonce := util.RandomString()
	opaque := util.RandomString()
	realm := "Digest realm=\"testrealm@host.com\", qop=\"auth,auth-int\", nonce=\"" + newNonce +
		"\", opaque=\"" + opaque + "\", algorithm=MD5, stale=FALSE"

	var authHeader string
	if vals := ex.Request.Header["Authorization"]; vals != nil && len(vals) == 1 {
		authHeader = vals[0]
	} else {
		ex.ResponseWriter.Header().Set(c.WWWAuthenticate, realm)
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
		ex.ResponseWriter.Header().Set(c.WWWAuthenticate, realm)
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		errMessage := err.Error()
		if errMessage == "http: named cookie not present" {
			errMessage = "Missing nonce cookie"
		}
		ex.WriteF("Error: %q\n", errMessage)
		return
	}

	if givenNonce != expectedNonce.Value {
		ex.ResponseWriter.Header().Set(c.WWWAuthenticate, realm)
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		ex.WriteF("Error: %q\nGiven: %q\nExpected: %q", "Nonce mismatch", givenNonce, expectedNonce.Value)
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
		ex.ResponseWriter.Header().Set(c.WWWAuthenticate, realm)
		http.SetCookie(ex.ResponseWriter, &http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
		ex.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		ex.WriteF("Error: %q\nGiven: %q\nExpected: %q", "Response code mismatch", givenResponseCode, expectedResponseCode)
		return
	}

	util.WriteJson(ex.ResponseWriter, map[string]any{
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
