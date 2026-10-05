package api_tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrailingSlashIsIgnored(t *testing.T) {
	for _, path := range []string{
		"get", "post", "headers", "any", "anything/foo", "payload", "ip", "ip.txt", "info", "health",
		"status/418", "delay/0", "bytes/0", "links/3", "range/10", "base64/aGk", "svg/bun", "cache", "cache/10",
		"etag/foo", "cookies", "cookies/set/a/b", "cookies/delete", "response-headers", "redirect/2",
		"relative-redirect/2", "absolute-redirect/2", "redirect-to?url=/get", "basic-auth/a/b", "bearer/x",
		"bearer", "digest-auth/a/b", "mix/s=201", "mixer", "help/mixer", "runner", "html", "robots.txt", "deny",
		"drip?delay=0&duration=0&numbytes=1", "sse?count=1",
	} {
		t.Run(path, func(t *testing.T) {
			withSlash := path + "/"
			if before, after, found := strings.Cut(path, "?"); found {
				withSlash = before + "/?" + after
			}
			resp, body := ExecRequest(t, R{Path: path})
			slashResp, slashBody := ExecRequest(t, R{Path: withSlash})
			if path != "bearer" { // Which is a 404 with usage help.
				assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
			}
			assert.Equal(t, resp.StatusCode, slashResp.StatusCode)
			if path == "svg/bun" || path == "links/3" || path == "status/418" {
				assert.Equal(t, body, slashBody)
			}
		})
	}
}

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
