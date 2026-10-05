package svg

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/ex"
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
			resp := ex.InvokeHandlerForTest(path, http.Request{}, `/svg/(?P<seed>.+)`, handleSVGSeeded)
			assert.Contains(t, resp.Body, text)
		})
	}
}
