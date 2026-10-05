package api_tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Standard base64 can end with a `/`, which is kept.
func TestTrailingSlashKeptInBase64(t *testing.T) {
	_, body := ExecRequest(t, R{Path: "base64/aGk/"})
	assert.Equal(t, "hi?", body)
}

func TestRelativeRedirectsWithTrailingSlash(t *testing.T) {
	for _, path := range []string{"relative-redirect/3/", "redirect/1/"} {
		t.Run(path, func(t *testing.T) {
			current, _ := url.Parse(BaseURL + path)
			for range 5 {
				resp, _ := ExecRequest(t, R{Path: strings.TrimPrefix(current.Path, "/") + "?" + current.RawQuery})
				if resp.StatusCode != http.StatusFound {
					break
				}
				location, _ := url.Parse(resp.Header.Get("Location"))
				current = current.ResolveReference(location)
			}
			assert.Equal(t, "/anything", current.Path)
		})
	}
}
