package oauth2

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
)

// authCodePayload is the data encoded in the authorization code
type authCodePayload struct {
	ClientID    string `json:"cid"`
	RedirectURI string `json:"uri"`
	State       string `json:"st,omitempty"`
	Email       string `json:"email"`
	CreatedAt   int64  `json:"iat"`
}

// accessTokenPayload is the data encoded in the access token
type accessTokenPayload struct {
	Email     string `json:"email"`
	ClientID  string `json:"cid"`
	CreatedAt int64  `json:"iat"`
}

var RouteList = []ex.Route{
	ex.NewRoute("/oauth2/authorize", handleAuthorize),
	ex.NewRoute("/oauth2/token", handleToken),
	ex.NewRoute("/oauth2/userinfo", handleUserinfo),
}

func handleAuthorize(ex *ex.Exchange) response.Response {
	if ex.Request.Method == http.MethodGet {
		return handleAuthorizeGet(ex)
	} else if ex.Request.Method == http.MethodPost {
		return handleAuthorizePost(ex)
	}
	return response.Response{
		Status: http.StatusMethodNotAllowed,
		Body:   "Method not allowed. Use GET or POST.",
	}
}

func handleAuthorizeGet(ex *ex.Exchange) response.Response {
	query := ex.Request.URL.Query()

	clientID := query.Get("client_id")
	if clientID == "" {
		return response.BadRequest("Missing required parameter: client_id")
	}

	redirectURI := query.Get("redirect_uri")
	if redirectURI == "" {
		return response.BadRequest("Missing required parameter: redirect_uri")
	}

	responseType := query.Get("response_type")
	if responseType != "code" {
		return response.BadRequest("Invalid or missing response_type. Only 'code' is supported.")
	}

	state := query.Get("state")
	scope := query.Get("scope")

	// Render the consent page
	return assets.Render("oauth2.html", *ex, map[string]any{
		"clientID":    clientID,
		"redirectURI": redirectURI,
		"state":       state,
		"scope":       scope,
	})
}

func handleAuthorizePost(ex *ex.Exchange) response.Response {
	if err := ex.Request.ParseForm(); err != nil {
		return response.BadRequest("Failed to parse form data")
	}

	clientID := ex.Request.FormValue("client_id")
	if clientID == "" {
		return response.BadRequest("Missing required parameter: client_id")
	}

	redirectURI := ex.Request.FormValue("redirect_uri")
	if redirectURI == "" {
		return response.BadRequest("Missing required parameter: redirect_uri")
	}

	state := ex.Request.FormValue("state")
	email := ex.Request.FormValue("email")
	decision := ex.Request.FormValue("decision")

	parsedURI, err := url.Parse(redirectURI)
	if err != nil {
		return response.BadRequest("Invalid redirect_uri")
	}

	q := parsedURI.Query()

	if decision == "approve" {
		if email == "" {
			return response.BadRequest("Missing required parameter: email")
		}

		// Create authorization code payload
		payload := authCodePayload{
			ClientID:    clientID,
			RedirectURI: redirectURI,
			State:       state,
			Email:       email,
			CreatedAt:   time.Now().Unix(),
		}

		// Encode as JSON then base64
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return response.BadRequest("Failed to generate authorization code")
		}
		code := base64.URLEncoding.EncodeToString(jsonBytes)

		q.Set("code", code)
		if state != "" {
			q.Set("state", state)
		}
	} else {
		// User denied access
		q.Set("error", "access_denied")
		q.Set("error_description", "The user denied the authorization request")
		if state != "" {
			q.Set("state", state)
		}
	}

	parsedURI.RawQuery = q.Encode()

	return response.Response{
		Status: http.StatusFound,
		Header: http.Header{
			"Location": []string{parsedURI.String()},
		},
	}
}

func handleToken(ex *ex.Exchange) response.Response {
	if ex.Request.Method != http.MethodPost {
		return response.Response{
			Status: http.StatusMethodNotAllowed,
			Body: map[string]any{
				"error":             "invalid_request",
				"error_description": "Method not allowed. Use POST.",
			},
		}
	}

	if err := ex.Request.ParseForm(); err != nil {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "invalid_request",
				"error_description": "Failed to parse form data",
			},
		}
	}

	grantType := ex.Request.FormValue("grant_type")
	if grantType != "authorization_code" {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "unsupported_grant_type",
				"error_description": "Only 'authorization_code' grant type is supported",
			},
		}
	}

	code := ex.Request.FormValue("code")
	if code == "" {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "invalid_request",
				"error_description": "Missing required parameter: code",
			},
		}
	}

	clientID := ex.Request.FormValue("client_id")
	if clientID == "" {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "invalid_request",
				"error_description": "Missing required parameter: client_id",
			},
		}
	}

	clientSecret := ex.Request.FormValue("client_secret")
	if clientSecret == "" {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "invalid_request",
				"error_description": "Missing required parameter: client_secret",
			},
		}
	}

	redirectURI := ex.Request.FormValue("redirect_uri")
	if redirectURI == "" {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "invalid_request",
				"error_description": "Missing required parameter: redirect_uri",
			},
		}
	}

	// Decode the authorization code
	jsonBytes, err := base64.URLEncoding.DecodeString(code)
	if err != nil {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "invalid_grant",
				"error_description": "Invalid authorization code format",
			},
		}
	}

	var codePayload authCodePayload
	if err := json.Unmarshal(jsonBytes, &codePayload); err != nil {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "invalid_grant",
				"error_description": "Invalid authorization code",
			},
		}
	}

	// Verify client_id matches
	if codePayload.ClientID != clientID {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "invalid_grant",
				"error_description": "client_id does not match the authorization request",
			},
		}
	}

	// Verify redirect_uri matches
	if codePayload.RedirectURI != redirectURI {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "invalid_grant",
				"error_description": "redirect_uri does not match the authorization request",
			},
		}
	}

	// Check if code is expired (10 minutes)
	createdAt := time.Unix(codePayload.CreatedAt, 0)
	if time.Since(createdAt) > 10*time.Minute {
		return response.Response{
			Status: http.StatusBadRequest,
			Body: map[string]any{
				"error":             "invalid_grant",
				"error_description": "Authorization code has expired",
			},
		}
	}

	// Create access token payload with email
	tokenPayload := accessTokenPayload{
		Email:     codePayload.Email,
		ClientID:  clientID,
		CreatedAt: time.Now().Unix(),
	}

	tokenBytes, err := json.Marshal(tokenPayload)
	if err != nil {
		return response.Response{
			Status: http.StatusInternalServerError,
			Body: map[string]any{
				"error":             "server_error",
				"error_description": "Failed to generate access token",
			},
		}
	}

	accessToken := base64.URLEncoding.EncodeToString(tokenBytes)

	return response.Response{
		Status: http.StatusOK,
		Header: http.Header{
			"Cache-Control": []string{"no-store"},
			"Pragma":        []string{"no-cache"},
		},
		Body: map[string]any{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   3600,
		},
	}
}

func handleUserinfo(ex *ex.Exchange) response.Response {
	// Accept GET or POST
	if ex.Request.Method != http.MethodGet && ex.Request.Method != http.MethodPost {
		return response.Response{
			Status: http.StatusMethodNotAllowed,
			Body: map[string]any{
				"error":             "invalid_request",
				"error_description": "Method not allowed. Use GET or POST.",
			},
		}
	}

	// Get the access token from Authorization header
	authHeader := ex.HeaderValueLast("Authorization")
	if authHeader == "" {
		return response.Response{
			Status: http.StatusUnauthorized,
			Header: http.Header{
				"WWW-Authenticate": []string{"Bearer"},
			},
			Body: map[string]any{
				"error":             "invalid_token",
				"error_description": "Missing Authorization header",
			},
		}
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return response.Response{
			Status: http.StatusUnauthorized,
			Header: http.Header{
				"WWW-Authenticate": []string{"Bearer"},
			},
			Body: map[string]any{
				"error":             "invalid_token",
				"error_description": "Invalid Authorization header format. Expected: Bearer <token>",
			},
		}
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Decode the access token
	jsonBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return response.Response{
			Status: http.StatusUnauthorized,
			Header: http.Header{
				"WWW-Authenticate": []string{"Bearer"},
			},
			Body: map[string]any{
				"error":             "invalid_token",
				"error_description": "Invalid access token format",
			},
		}
	}

	var tokenPayload accessTokenPayload
	if err := json.Unmarshal(jsonBytes, &tokenPayload); err != nil {
		return response.Response{
			Status: http.StatusUnauthorized,
			Header: http.Header{
				"WWW-Authenticate": []string{"Bearer"},
			},
			Body: map[string]any{
				"error":             "invalid_token",
				"error_description": "Invalid access token",
			},
		}
	}

	// Check if token is expired (1 hour)
	createdAt := time.Unix(tokenPayload.CreatedAt, 0)
	if time.Since(createdAt) > time.Hour {
		return response.Response{
			Status: http.StatusUnauthorized,
			Header: http.Header{
				"WWW-Authenticate": []string{"Bearer"},
			},
			Body: map[string]any{
				"error":             "invalid_token",
				"error_description": "Access token has expired",
			},
		}
	}

	return response.Response{
		Status: http.StatusOK,
		Body: map[string]any{
			"email": tokenPayload.Email,
		},
	}
}
