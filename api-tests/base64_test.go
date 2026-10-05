package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBase64(t *testing.T) {
	for path, want := range map[string]string{
		"base64":          "HTTPBUN is awesomer!",
		"base64/aGk=":     "hi",
		"base64/aGk":      "hi",
		"b64/aGk/Pg==":    "hi?>",
		"base64/aGk_Pg==": "hi?>",
		"base64/aGk_Pg":   "hi?>",
		"base64/Pz8%2B":   "??>",
		"base64/Pz8+":     "??>",
		"base64/Pz8-":     "??>",
	} {
		t.Run(path, func(t *testing.T) {
			resp, body := ExecRequest(R{Path: path})
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, want, body)
		})
	}
}

func TestBase64Invalid(t *testing.T) {
	for _, path := range []string{"base64/a", "base64/a!b=", "base64/aGk=x"} {
		t.Run(path, func(t *testing.T) {
			resp, _ := ExecRequest(R{Path: path})
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}
