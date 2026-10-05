package api_tests

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sharat87/httpbun/server"
	"github.com/sharat87/httpbun/server/spec"
)

// Requests are always sent to this host, whichever port the test server is actually listening on. So URLs and the
// `Host` header in responses are the same on every run.
const (
	Host    = "httpbun.test"
	BaseURL = "http://" + Host + "/"
)

// TestServer is an httpbun server for tests, on a random port, so tests can run in parallel with other test runs.
type TestServer struct {
	client *http.Client
}

type R struct {
	Method  string
	Path    string
	Body    string
	Headers map[string][]string
}

var defaultServer *TestServer

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)

	ts := httptest.NewServer(server.New(spec.Spec{}))
	defaultServer = newTestServer(ts)
	code := m.Run()
	ts.Close()
	os.Exit(code)
}

// NewServer starts a server with the given spec, for tests that need a configuration other than the default.
func NewServer(t *testing.T, s spec.Spec) *TestServer {
	ts := httptest.NewServer(server.New(s))
	t.Cleanup(ts.Close)
	return newTestServer(ts)
}

func newTestServer(ts *httptest.Server) *TestServer {
	addr := ts.Listener.Addr().String()
	dialer := &net.Dialer{}
	return &TestServer{
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
					return dialer.DialContext(ctx, network, addr)
				},
			},
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
	}
}

// ExecRequest sends a request to the default test server.
func ExecRequest(t *testing.T, r R) (http.Response, string) {
	t.Helper()
	return defaultServer.Exec(t, r)
}

// Exec sends a request, and returns the response with its body read. Failing to get a response fails the test.
func (s *TestServer) Exec(t *testing.T, r R) (http.Response, string) {
	t.Helper()

	var bodyReader io.Reader
	if r.Body != "" {
		bodyReader = strings.NewReader(r.Body)
	}

	if r.Method == "" {
		r.Method = http.MethodGet
	}

	req, err := http.NewRequest(r.Method, BaseURL+r.Path, bodyReader)
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}

	req.Header.Set("User-Agent", "")
	for name, values := range r.Headers {
		req.Header[name] = values
	}

	resp, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	return *resp, string(bodyText)
}
