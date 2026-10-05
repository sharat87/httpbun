package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

const oauthRedirectURI = "https://example.com/callback"

// oauthAuthorize approves the consent form, and returns the authorization code from the redirect.
func oauthAuthorize(t *testing.T) string {
	form := url.Values{
		"client_id":    {"my-app"},
		"redirect_uri": {oauthRedirectURI},
		"state":        {"abc"},
		"email":        {"user@example.com"},
		"decision":     {"approve"},
	}
	resp, _ := ExecRequest(R{
		Method:  http.MethodPost,
		Path:    "oauth2/authorize",
		Body:    form.Encode(),
		Headers: map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
	})
	if !assert.Equal(t, http.StatusFound, resp.StatusCode) {
		return ""
	}
	location, _ := url.Parse(resp.Header.Get("Location"))
	assert.Equal(t, "abc", location.Query().Get("state"))
	return location.Query().Get("code")
}

func oauthToken(form url.Values, headers map[string][]string) (http.Response, map[string]any) {
	if headers == nil {
		headers = map[string][]string{}
	}
	headers["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	resp, body := ExecRequest(R{Method: http.MethodPost, Path: "oauth2/token", Body: form.Encode(), Headers: headers})
	var data map[string]any
	_ = json.Unmarshal([]byte(body), &data)
	return resp, data
}

func TestOAuth2CodeFlow(t *testing.T) {
	for _, mode := range []string{"client_secret_post", "client_secret_basic"} {
		t.Run(mode, func(t *testing.T) {
			s := assert.New(t)
			code := oauthAuthorize(t)

			form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {oauthRedirectURI}}
			headers := map[string][]string{}
			if mode == "client_secret_post" {
				form.Set("client_id", "my-app")
				form.Set("client_secret", "secret")
			} else {
				headers["Authorization"] = []string{"Basic " + base64.StdEncoding.EncodeToString([]byte("my-app:secret"))}
			}

			resp, token := oauthToken(form, headers)
			if !s.Equal(http.StatusOK, resp.StatusCode, token) {
				return
			}
			accessToken, _ := token["access_token"].(string)
			s.NotEmpty(accessToken)

			resp, body := ExecRequest(R{
				Path:    "oauth2/userinfo",
				Headers: map[string][]string{"Authorization": {"bearer " + accessToken}},
			})
			s.Equal(http.StatusOK, resp.StatusCode)
			s.Contains(body, "user@example.com")
		})
	}
}

func TestOAuth2TokenWrongClient(t *testing.T) {
	code := oauthAuthorize(t)
	resp, data := oauthToken(url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {oauthRedirectURI},
		"client_id":     {"other-app"},
		"client_secret": {"secret"},
	}, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid_grant", data["error"])
}
