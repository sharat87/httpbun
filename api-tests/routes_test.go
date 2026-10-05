package api_tests

import (
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/routes"
	"github.com/sharat87/httpbun/server/spec"
)

// endpoint is an example request for a route, with where it's documented. Every route must have at least one, so
// adding a route without documenting it fails TestEveryRouteIsDocumented.
type endpoint struct {
	// An example path, with an optional query string, without the leading slash.
	path string
	// Where it's documented: `#anchor` on the homepage, or the path of a page like `/runner`.
	docs string
	// Why it's not documented, if docs is empty.
	undocumented string
	// The example is expected to be a 404, like an endpoint that needs more path segments.
	notFound bool
}

var endpoints = []endpoint{
	{path: "", docs: "/"},
	{path: "index.html", docs: "/"},
	{path: "health", undocumented: "health check for deployments"},
	{path: "info", docs: "#configuration-info"},
	{path: "assets/icon-32.png", undocumented: "static files for the HTML pages"},

	{path: "get", docs: "#get"},
	{path: "post", docs: "#post"},
	{path: "put", docs: "#put"},
	{path: "patch", docs: "#patch"},
	{path: "delete", docs: "#delete"},
	{path: "any", docs: "#any"},
	{path: "anything/foo?a=1", docs: "#any-extra-path"},
	{path: "headers", docs: "#headers"},
	{path: "payload", docs: "#payload"},

	{path: "mix/s=201/h=x-a:b", docs: "#mix-endpoint"},
	{path: "mixer", docs: "/help/mixer"},
	{path: "help/mixer", docs: "/help/mixer"},
	{path: "runner", docs: "/runner"},
	{path: "run/cmV0dXJuIHt9", docs: "/runner"},

	{path: "basic-auth/user/pass", docs: "#basic-auth"},
	{path: "bearer", docs: "#bearer-token", notFound: true},
	{path: "bearer/token", docs: "#bearer-token"},
	{path: "digest-auth/user/pass", docs: "#digest-auth"},
	{path: "digest-auth/auth/user/pass", docs: "#digest-auth-qop"},
	{path: "oauth2/authorize?client_id=app&redirect_uri=https://example.com/cb", docs: "#oauth2-authorize"},
	{path: "oauth2/token", docs: "#oauth2-token"},
	{path: "oauth2/userinfo", docs: "#oauth2-userinfo"},

	{path: "ip", docs: "#ip"},
	{path: "ip.json", docs: "#ip-json"},
	{path: "ip.txt", docs: "#ip-txt"},
	{path: "cache", docs: "#cache"},
	{path: "cache/60", docs: "#cache-aged"},
	{path: "etag/abc", docs: "#etag"},
	{path: "status/418", docs: "#status"},
	{path: "response-headers?x-a=b", docs: "#response-headers"},
	{path: "respond-with-headers?x-a=b", docs: "#respond-with-headers"},
	{path: "deny", docs: "#deny"},
	{path: "html", docs: "#html"},
	{path: "svg/bun", docs: "#svg"},
	{path: "robots.txt", docs: "#robots"},
	{path: "base64", docs: "#base64"},
	{path: "base64/aGk=", docs: "#base64-with-input"},
	{path: "b64/aGk=", docs: "#base64-with-input"},
	{path: "bytes/0", docs: "#bytes"},
	{path: "delay/0", docs: "#delay"},
	{path: "drip?delay=0&duration=0&numbytes=1", docs: "#drip"},
	{path: "drip-lines?delay=0&duration=0&numbytes=1", docs: "#drip-lines"},
	{path: "sse?count=1", docs: "#sse"},
	{path: "links/3", docs: "#links"},
	{path: "links/3/1", docs: "#links-offset"},
	{path: "range/10", docs: "#range"},

	{path: "cookies", docs: "#cookies"},
	{path: "cookie", docs: "#cookies"},
	{path: "cookies/set?a=b", docs: "#cookies-set-query"},
	{path: "cookies/set/a/b", docs: "#cookies-set-path"},
	{path: "cookies/delete?a", docs: "#cookies-delete"},

	{path: "redirect?url=https://example.com/", docs: "#redirect"},
	{path: "redirect-to?url=https://example.com/", docs: "#redirect-to"},
	{path: "redirect/2", docs: "#redirect-count"},
	{path: "relative-redirect/2", docs: "#relative-redirect"},
	{path: "absolute-redirect/2", docs: "#absolute-redirect"},

	{path: "llm/v1/chat/completions", docs: "#llm-chat-completions"},
	{path: "llm/v1/completions", docs: "#llm-completions"},
	{path: "llm/v1/responses", docs: "#llm-responses"},
	{path: "llm/v1/messages", docs: "#llm-messages"},
	{path: "llm/chat/completions", undocumented: "backwards compatible path, without /v1"},
	{path: "llm/completions", undocumented: "backwards compatible path, without /v1"},
	{path: "llm/responses", undocumented: "backwards compatible path, without /v1"},
	{path: "llm/messages", undocumented: "backwards compatible path, without /v1"},
}

// routeIndex is the index of the route that serves the path, which is the first one that matches, or -1.
func routeIndex(path string) int {
	path, _, _ = strings.Cut(path, "?")
	for i, route := range routes.GetRoutes() {
		if route.Pat.MatchString("/" + path) {
			return i
		}
	}
	return -1
}

func TestEveryRouteIsDocumented(t *testing.T) {
	covered := map[int]bool{}
	for _, e := range endpoints {
		i := routeIndex(e.path)
		if assert.NotEqual(t, -1, i, "no route matches example %q", e.path) {
			covered[i] = true
		}
		assert.True(t, e.docs != "" || e.undocumented != "", "%q needs docs, or a reason it's undocumented", e.path)
	}

	for i, route := range routes.GetRoutes() {
		assert.True(t, covered[i], "route %s has no example in `endpoints`. Add one, and document the route.", route.Pat.String())
	}
}

var endpointDtPattern = regexp.MustCompile(`<dt(?: id=([\w-]+))?>(/[^<]*)`)

func TestEveryDocumentedEndpointExists(t *testing.T) {
	_, homepage := ExecRequest(t, R{Path: ""})

	referenced := map[string]bool{}
	for _, e := range endpoints {
		if anchor, isAnchor := strings.CutPrefix(e.docs, "#"); isAnchor {
			referenced[anchor] = true
			assert.Contains(t, homepage, " id="+anchor+">", "%q is documented at a missing anchor", e.path)
		}
	}

	for _, m := range endpointDtPattern.FindAllStringSubmatch(homepage, -1) {
		anchor, path := m[1], m[2]
		if assert.NotEmpty(t, anchor, "documented endpoint %q needs an id on its <dt>", path) {
			assert.True(t, referenced[anchor], "documented endpoint %q (#%s) has no example in `endpoints`", path, anchor)
		}
	}
}

// TestEndpointsSmoke requests every example, with and without a path prefix and a trailing slash, and checks they all
// respond the same, without a server error.
func TestEndpointsSmoke(t *testing.T) {
	prefixed := NewServer(t, spec.Spec{PathPrefix: "/mount"})

	for _, e := range endpoints {
		t.Run(e.path, func(t *testing.T) {
			resp, body := ExecRequest(t, R{Path: e.path})
			assert.Less(t, resp.StatusCode, 500, body)
			if e.notFound {
				assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			} else {
				assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, body)
			}

			prefixedResp, _ := prefixed.Exec(t, R{Path: "mount/" + e.path})
			assert.Equal(t, resp.StatusCode, prefixedResp.StatusCode, "with a path prefix")

			if location := prefixedResp.Header.Get("Location"); strings.HasPrefix(location, "/") {
				assert.True(t, strings.HasPrefix(location, "/mount/"), "redirect %q is outside the path prefix", location)
			}

			if path, query, _ := strings.Cut(e.path, "?"); path != "" && !strings.HasSuffix(path, "=") {
				slashResp, _ := ExecRequest(t, R{Path: path + "/?" + query})
				assert.Equal(t, resp.StatusCode, slashResp.StatusCode, "with a trailing slash")
			}
		})
	}
}

var linkPattern = regexp.MustCompile(`(?:href|src)=["']?([^"' >]+)`)

// TestPageLinksWithPathPrefix checks that links on the HTML pages work with a path prefix, which is easy to break by
// writing a link starting with `/`.
func TestPageLinksWithPathPrefix(t *testing.T) {
	prefixed := NewServer(t, spec.Spec{PathPrefix: "/mount"})

	for _, page := range []string{"mount", "mount/", "mount/mixer", "mount/runner", "mount/help/mixer"} {
		t.Run(page, func(t *testing.T) {
			_, html := prefixed.Exec(t, R{Path: page})
			pageURL, _ := url.Parse(BaseURL + page)

			var links []string
			for _, m := range linkPattern.FindAllStringSubmatch(html, -1) {
				// Like `//mount`, from writing `/{{.pathPrefix}}`, which points to a host named `mount`.
				assert.False(t, strings.HasPrefix(m[1], "//"), "protocol-relative link %q", m[1])
				link, _ := url.Parse(m[1])
				target := pageURL.ResolveReference(link)
				if target.Host != Host || slices.Contains(links, target.Path) {
					continue
				}
				links = append(links, target.Path)
				resp, _ := prefixed.Exec(t, R{Path: strings.TrimPrefix(target.Path, "/")})
				assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "link %q resolves to %q", m[1], target.Path)
			}
			assert.NotEmpty(t, links)
		})
	}
}
