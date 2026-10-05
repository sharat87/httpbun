package run

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/ex"
)

func runScript(script string) (int, http.Header, any) {
	resp := ex.InvokeHandlerForTest(
		"run/"+base64.URLEncoding.EncodeToString([]byte(script)),
		http.Request{Method: http.MethodGet, Header: http.Header{}},
		`/run`+restPathPattern,
		handleRunJS,
	)
	return resp.Status, resp.Header, resp.Body
}

func TestRunBasic(t *testing.T) {
	s := assert.New(t)
	status, _, body := runScript(`return {status: 201, body: "hi " + R.method}`)
	s.Equal(201, status)
	s.Equal([]byte("hi GET"), body)
}

func TestRunHeaderList(t *testing.T) {
	s := assert.New(t)
	status, headers, _ := runScript(`return {headers: {"X-A": ["1", "2"], "X-B": "3"}}`)
	s.Equal(200, status)
	s.Equal([]string{"1", "2"}, headers.Values("X-A"))
	s.Equal("3", headers.Get("X-B"))
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
			status, _, _ := runScript(script)
			assert.Equal(t, http.StatusBadRequest, status)
		})
	}
}
