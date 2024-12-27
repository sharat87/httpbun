package auth

import (
	"fmt"
	"github.com/sharat87/httpbun/response"
	"net/http"
	"regexp"
	"strings"

	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/util"
)

var Routes = map[string]exchange.HandlerFn2{
	"/basic-auth/(?P<user>[^/]+)/(?P<pass>[^/]+)/?":                                       handleAuthBasic,
	"/bearer(/(?P<tok>\\w+))?":                                                            handleAuthBearer,
	"/digest-auth/(?P<qop>auth|auth-int|auth,auth-int)/(?P<user>[^/]+)/(?P<pass>[^/]+)/?": handleAuthDigest,
	"/digest-auth/(?P<user>[^/]+)/(?P<pass>[^/]+)/?":                                      handleAuthDigest,
}

func handleAuthBasic(ex *exchange.Exchange) response.Response {
	givenUsername, givenPassword, ok := ex.Request.BasicAuth()
	isAuthenticated := ok && givenUsername == ex.Field("user") && givenPassword == ex.Field("pass")

	if !isAuthenticated {
		return response.New(http.StatusUnauthorized, http.Header{
			c.WWWAuthenticate: []string{"Basic realm=\"Fake Realm\""},
		}, nil)
	}

	return response.JSON(http.StatusOK, nil, map[string]any{
		"authenticated": isAuthenticated,
		"user":          givenUsername,
	})
}

func handleAuthBearer(ex *exchange.Exchange) response.Response {
	expectedToken := ex.Field("tok")

	authHeader := ex.HeaderValueLast("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return response.New(http.StatusUnauthorized, http.Header{
			c.WWWAuthenticate: []string{"Bearer"},
		}, nil)
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	return response.JSON(http.StatusOK, nil, map[string]any{
		"authenticated": token != "" && (expectedToken == "" || expectedToken == token),
		"token":         token,
	})
}

func handleAuthDigest(ex *exchange.Exchange) response.Response {
	expectedQop, expectedUsername, expectedPassword := ex.Field("qop"), ex.Field("user"), ex.Field("pass")

	requireCookieParamValue, _ := ex.QueryParamSingle("require-cookie")
	requireCookie := requireCookieParamValue == "true" || requireCookieParamValue == "1" || requireCookieParamValue == "t"

	var authHeader string
	if vals := ex.Request.Header["Authorization"]; len(vals) == 1 {
		authHeader = vals[0]
	} else {
		return unauthorizedDigest(expectedQop, requireCookie, "")
	}

	givenDetails := parseDigestAuthHeader(authHeader)

	// QOP check.
	if expectedQop != "" && givenDetails["qop"] != expectedQop {
		return unauthorizedDigest(expectedQop, requireCookie, fmt.Sprintf("Error: %q\n", "Unsupported QOP"))
	}

	// Nonce check.
	givenNonce := givenDetails["nonce"]

	if requireCookie {
		expectedNonce, err := ex.Request.Cookie("nonce")
		if err != nil {
			errMessage := err.Error()
			if errMessage == "http: named cookie not present" {
				errMessage = "Missing nonce cookie"
			}

			return unauthorizedDigest(expectedQop, requireCookie, fmt.Sprintf("Error: %q\n", errMessage))
		}

		if givenNonce != expectedNonce.Value {
			msg := fmt.Sprintf("Error: %q\nGiven: %q\nExpected: %q", "Nonce mismatch", givenNonce, expectedNonce.Value)
			return unauthorizedDigest(expectedQop, requireCookie, msg)
		}
	}

	// Response code check.
	expectedResponseCode, err := computeDigestAuthResponse(
		expectedUsername,
		expectedPassword,
		givenNonce,
		givenDetails["nc"],
		givenDetails["cnonce"],
		expectedQop,
		ex,
	)
	if err != nil {
		return unauthorizedDigest(expectedQop, requireCookie, fmt.Sprintf("Error: %q\n", err.Error()))
	}

	givenResponseCode := givenDetails["response"]

	if expectedResponseCode != givenResponseCode {
		msg := fmt.Sprintf("Error: %q\nGiven: %q\nExpected: %q", "Response code mismatch", givenResponseCode, expectedResponseCode)
		return unauthorizedDigest(expectedQop, requireCookie, msg)
	}

	return response.Response{
		Body: map[string]any{
			"authenticated": true,
			"user":          expectedUsername,
		},
	}
}

// unauthorizedDigest builds a response with status 401 Unauthorized and WWW-Authenticate header, for Digest auth.
func unauthorizedDigest(expectedQop string, setCookie bool, body string) response.Response {
	qop := expectedQop
	if qop == "" {
		qop = "auth,auth-int"
	}

	newNonce := util.RandomString()
	opaque := util.RandomString()

	var cookies []http.Cookie
	if setCookie {
		cookies = append(cookies, http.Cookie{
			Name:  "nonce",
			Value: newNonce,
		})
	}

	return response.Response{
		Status: http.StatusUnauthorized,
		Header: http.Header{c.WWWAuthenticate: []string{
			"Digest realm=\"testrealm@host.com\", qop=\"" + qop + "\", nonce=\"" + newNonce +
				"\", opaque=\"" + opaque + "\", algorithm=MD5, stale=FALSE",
		}},
		Cookies: cookies,
		Body:    body,
	}
}

func parseDigestAuthHeader(header string) map[string]string {
	matches := regexp.MustCompile("([a-z]+)=(?:\"([^\"]+)\"|([^,]+))").FindAllStringSubmatch(header, -1)
	givenDetails := make(map[string]string)

	for _, m := range matches {
		key := m[1]
		val := m[2]
		if val == "" {
			val = m[3]
		}
		givenDetails[key] = val
	}

	return givenDetails
}

// Digest auth response computer.
func computeDigestAuthResponse(username, password, serverNonce, nc, clientNonce, qop string, ex *exchange.Exchange) (string, error) {
	method := ex.Request.Method
	path := ex.Request.URL.Path
	entityBody := ex.BodyString()

	// Source: <https://en.wikipedia.org/wiki/Digest_access_authentication>.
	if qop != "" && qop != "auth" && qop != "auth-int" {
		return "", fmt.Errorf("unsupported qop: %q", qop)
	}

	ha1 := util.Md5sum(username + ":" + "testrealm@host.com" + ":" + password)

	var ha2 string
	if qop == "" || qop == "auth" {
		ha2 = util.Md5sum(method + ":" + path)
	} else {
		ha2 = util.Md5sum(method + ":" + path + ":" + util.Md5sum(entityBody))
	}

	if qop == "" {
		return util.Md5sum(ha1 + ":" + serverNonce + ":" + ha2), nil
	}

	return util.Md5sum(ha1 + ":" + serverNonce + ":" + nc + ":" + clientNonce + ":" + qop + ":" + ha2), nil
}
