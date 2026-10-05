package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/server/spec"
)

func TestLinksWithPathPrefix(t *testing.T) {
	s := assert.New(t)
	req := httptest.NewRequest(http.MethodGet, "http://localhost/mount/links/2", nil)
	exchange := ex.New(nil, req, spec.Spec{PathPrefix: "/mount"})
	s.True(exchange.MatchAndLoadFields(ex.MakePat("/links/(?P<count>\\d+)(/(?P<offset>\\d+))?/?")))
	resp := handleLinks(exchange)
	s.Contains(resp.Body, "<a href='/mount/links/2/1'>1</a>")
}
