package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSVGText(t *testing.T) {
	for path, text := range map[string]string{
		"svg/bun":                ">BU</text>",
		"svg/a":                  ">A</text>",
		"svg/%C3%A9t%C3%A9":      ">ÉT</text>",
		"svg/%E6%97%A5%E6%9C%AC": ">日本</text>",
		"svg/%3Cscript%3E":       ">&lt;S</text>",
	} {
		t.Run(path, func(t *testing.T) {
			resp, body := ExecRequest(t, R{Path: path})
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "image/svg+xml", resp.Header.Get("Content-Type"))
			assert.Contains(t, body, "<svg ")
			assert.Contains(t, body, text)
		})
	}
}
