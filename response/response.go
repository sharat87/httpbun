package response

import (
	"fmt"
	"net/http"
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

type BodyWriter struct {
	w http.ResponseWriter
}

func NewBodyWriter(w http.ResponseWriter) BodyWriter {
	return BodyWriter{w}
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
