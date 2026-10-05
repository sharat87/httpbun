package api_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func offsets(start, end int) string {
	b := make([]byte, 0, end-start+1)
	for i := start; i <= end; i++ {
		b = append(b, byte(i%256))
	}
	return string(b)
}

func TestRangeFullBody(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{Path: "range/300"})
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("bytes", resp.Header.Get("Accept-Ranges"))
	s.Equal(offsets(0, 299), body)
	s.Equal(byte(255), body[255])
	s.Equal(byte(0), body[256])
}

func TestRangePartial(t *testing.T) {
	for _, tt := range []struct {
		header       string
		start, end   int
		contentRange string
	}{
		{"bytes=10-19", 10, 19, "bytes 10-19/300"},
		{"bytes=250-260", 250, 260, "bytes 250-260/300"},
		{"bytes=290-", 290, 299, "bytes 290-299/300"},
		{"bytes=-5", 295, 299, "bytes 295-299/300"},
		{"bytes=290-999", 290, 299, "bytes 290-299/300"},
		{"bytes=-999", 0, 299, "bytes 0-299/300"},
	} {
		t.Run(tt.header, func(t *testing.T) {
			resp, body := ExecRequest(t, R{Path: "range/300", Headers: map[string][]string{"Range": {tt.header}}})
			assert.Equal(t, http.StatusPartialContent, resp.StatusCode)
			assert.Equal(t, tt.contentRange, resp.Header.Get("Content-Range"))
			assert.Equal(t, offsets(tt.start, tt.end), body)
		})
	}
}

func TestRangeNotSatisfiable(t *testing.T) {
	s := assert.New(t)
	resp, body := ExecRequest(t, R{Path: "range/300", Headers: map[string][]string{"Range": {"bytes=300-"}}})
	s.Equal(http.StatusRequestedRangeNotSatisfiable, resp.StatusCode)
	s.Equal("bytes */300", resp.Header.Get("Content-Range"))
	s.Empty(body)
}

func TestRangeIgnoredHeaders(t *testing.T) {
	for _, header := range []string{"bytes=0-1,5-6", "bytes=5-2", "items=0-1", "bytes=-", "nonsense"} {
		t.Run(header, func(t *testing.T) {
			resp, body := ExecRequest(t, R{Path: "range/20", Headers: map[string][]string{"Range": {header}}})
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, offsets(0, 19), body)
		})
	}
}
