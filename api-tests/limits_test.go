package api_tests

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sharat87/httpbun/server/spec"
)

func TestConcurrencyLimit(t *testing.T) {
	s := assert.New(t)
	srv := NewServer(t, spec.Spec{MaxConcurrentRequests: 1})

	// Hold the only slot with a slow request, which stops when it's cancelled. It's retried if it gets a 503, which it
	// can, if it comes in while one of the requests below is being handled.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, BaseURL+"delay/10", nil)
	s.NoError(err)
	go func() {
		for {
			resp, err := srv.client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusServiceUnavailable {
				return
			}
		}
	}()

	resp, body := waitForStatus(t, srv, "get", http.StatusServiceUnavailable)
	s.Equal(http.StatusServiceUnavailable, resp.StatusCode)
	s.Equal("5", resp.Header.Get("Retry-After"))
	s.Equal("Server is busy, please try again later.\n", body)

	resp, body = srv.Exec(t, R{Path: "health"})
	s.Equal(http.StatusOK, resp.StatusCode, "health checks skip the limit")
	s.Equal("ok", body)

	// When the slow request's client goes away, its slot is freed.
	cancel()
	resp, _ = waitForStatus(t, srv, "get", http.StatusOK)
	s.Equal(http.StatusOK, resp.StatusCode)
}

// waitForStatus sends requests to the path until one gets the given status, or a few seconds pass, and returns the last
// response.
func waitForStatus(t *testing.T, srv *TestServer, path string, status int) (http.Response, string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, body := srv.Exec(t, R{Path: path})
		if resp.StatusCode == status || time.Now().After(deadline) {
			return resp, body
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBodySizeLimit(t *testing.T) {
	srv := NewServer(t, spec.Spec{MaxBodySize: 100})

	for _, path := range []string{"post", "payload", "mix"} {
		t.Run(path, func(t *testing.T) {
			s := assert.New(t)

			resp, _ := srv.Exec(t, R{Method: http.MethodPost, Path: path, Body: strings.Repeat("a", 100)})
			s.Equal(http.StatusOK, resp.StatusCode)

			resp, body := srv.Exec(t, R{Method: http.MethodPost, Path: path, Body: strings.Repeat("a", 101)})
			s.Equal(http.StatusRequestEntityTooLarge, resp.StatusCode)
			s.Equal("Request body is too large, the limit is 100 bytes.\n", body)
		})
	}
}

func TestBodySizeLimitChunked(t *testing.T) {
	srv := NewServer(t, spec.Spec{MaxBodySize: 100})

	for _, tc := range []struct {
		path        string
		contentType string
		body        string
	}{
		{"post", "text/plain", strings.Repeat("a", 101)},
		{"post", "application/json", `"` + strings.Repeat("a", 101) + `"`},
		{"post", "application/x-www-form-urlencoded", "a=" + strings.Repeat("a", 101)},
		{"post", "multipart/form-data; boundary=xyz", "--xyz\r\nContent-Disposition: form-data; name=\"a\"\r\n\r\n" +
			strings.Repeat("a", 101) + "\r\n--xyz--\r\n"},
		{"payload", "text/plain", strings.Repeat("a", 101)},
	} {
		t.Run(tc.path+" "+tc.contentType, func(t *testing.T) {
			s := assert.New(t)

			// Without a known length, the body is sent chunked, and the limit is only hit while reading it.
			req, err := http.NewRequest(http.MethodPost, BaseURL+tc.path, io.MultiReader(strings.NewReader(tc.body)))
			s.NoError(err)
			req.Header.Set("Content-Type", tc.contentType)
			s.Equal(int64(0), req.ContentLength, "length is unknown")
			resp, err := srv.client.Do(req)
			if !s.NoError(err) {
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			s.NoError(err)
			s.Equal(http.StatusRequestEntityTooLarge, resp.StatusCode)
			s.Equal("Request body is too large, the limit is 100 bytes.\n", string(body))
		})
	}
}
