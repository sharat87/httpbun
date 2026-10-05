package api_tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinks(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{Path: "links/3/1"})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Contains(body, "<a href='/links/3/0'>0</a> 1 <a href='/links/3/2'>2</a>")
}

func TestLinksCountIsCapped(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{Path: "links/999999999999999999999"})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(199, strings.Count(body, "<a href=")) // offset defaults to 0, which is not a link
	s.Contains(body, "<a href='/links/200/199'>199</a>")
}
