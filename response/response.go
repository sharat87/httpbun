package response

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sharat87/httpbun/util"
)

type Response struct {
	Status  int
	Header  http.Header
	Cookies []http.Cookie
	Body    any
	Writer  func(w BodyWriter)
}

func New(status int, header http.Header, body []byte) Response {
	return Response{
		Status: status,
		Header: header,
		Body:   body,
	}
}

func BadRequest(message string, vars ...any) Response {
	return Response{
		Status: http.StatusBadRequest,
		Body:   fmt.Sprintf(message, vars...),
	}
}

// How long clients are asked to wait before retrying, when the server is busy.
const RetryAfterSeconds = "5"

// Busy is a 503 for a request that can't be handled now because the server is busy, with the given message.
func Busy(message string) Response {
	return Response{
		Status: http.StatusServiceUnavailable,
		Header: http.Header{"Retry-After": {RetryAfterSeconds}},
		Body:   message,
	}
}

// LongBusy is a 503 for a request that can take long, when the server is handling as many of those as it can.
func LongBusy() Response {
	return Busy("Server is busy with too many long-running requests, like /delay, /drip and streams, please try again later.")
}

type BodyWriter struct {
	w   http.ResponseWriter
	ctx context.Context
}

// NewBodyWriter makes a BodyWriter for a response, with the request's context, which is done when the client goes away.
func NewBodyWriter(ctx context.Context, w http.ResponseWriter) BodyWriter {
	return BodyWriter{w: w, ctx: ctx}
}

// Sleep waits between writes. It returns false if the client went away, and the writer should stop.
func (bw BodyWriter) Sleep(d time.Duration) bool {
	return util.Sleep(bw.ctx, d)
}

func (bw BodyWriter) Write(content string) error {
	_, err := bw.w.Write([]byte(content))
	if err != nil {
		return err
	}

	f, ok := bw.w.(http.Flusher)
	if ok {
		f.Flush()
	} else {
		return fmt.Errorf("flush not available")
	}

	return nil
}
