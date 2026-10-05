package api_tests

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func runPath(script string) string {
	return "run/" + base64.URLEncoding.EncodeToString([]byte(script))
}

func TestRunBasic(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{Path: runPath(`return {status: 201, body: "hi " + R.method}`)})
	s.Equal(http.StatusCreated, resp.StatusCode)
	s.Equal("hi GET", body)
}

func TestRunHeaderList(t *testing.T) {
	s := assert.New(t)
	resp, _ := ExecRequest(t, R{Path: runPath(`return {headers: {"X-A": ["1", "2"], "X-B": "3"}}`)})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal([]string{"1", "2"}, resp.Header.Values("X-A"))
	s.Equal([]string{"3"}, resp.Header.Values("X-B"))
}

func TestRunInvalidScripts(t *testing.T) {
	for _, script := range []string{
		`let x = 1`,
		`return "hi"`,
		`return null`,
		`return {status: 42}`,
		`return {status: 100}`,
		`return {status: 1000}`,
		`return {status: -1}`,
		`return {headers: {"X-A": [1]}}`,
	} {
		t.Run(script, func(t *testing.T) {
			resp, body := ExecRequest(t, R{Path: runPath(script)})
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
			assert.NotEmpty(t, body)
		})
	}
}

func TestRunRequestHeaders(t *testing.T) {
	s := assert.New(t)
	script := `return {body: JSON.stringify({
		keys: Object.keys(R.headers).sort().join(","),
		multi: R.headers["x-multi"],
		missing: R.headers["X-Multi"] === undefined,
	})}`
	resp, body := ExecRequest(t, R{
		Path:    runPath(script),
		Headers: map[string][]string{"X-Multi": {"1", "2"}, "X-Other": {"o"}},
	})
	s.Equal(http.StatusOK, resp.StatusCode)
	// Go's HTTP client adds `Accept-Encoding`, and the server restores `Host` into the request headers.
	s.JSONEq(`{"keys": "accept-encoding,host,x-multi,x-other", "multi": "1, 2", "missing": true}`, body)
}

func TestRunRequestIsReadOnly(t *testing.T) {
	for _, script := range []string{
		`R.headers["x-new"] = "1"`,
		`R.headers["x-multi"] = "1"`,
		`delete R.headers["x-multi"]`,
		`Object.defineProperty(R.headers, "x-new", {value: "1"})`,
		`R.method = "POST"`,
		`R.headers = {}`,
		`delete R.extraPath`,
	} {
		t.Run(script, func(t *testing.T) {
			resp, body := ExecRequest(t, R{Path: runPath(script + "; return {}")})
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Contains(t, body, "read-only")
		})
	}
}
