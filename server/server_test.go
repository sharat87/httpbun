package server

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
)

func TestPanicBecomesServerError(t *testing.T) {
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	s := Server{routes: []ex.Route{
		ex.NewRoute("/boom", func(_ *ex.Exchange) response.Response {
			panic("something broke")
		}),
	}}
	ts := httptest.NewServer(s)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/boom")
	if !assert.NoError(t, err) {
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, string(body), "something broke")
}
