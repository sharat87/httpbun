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
