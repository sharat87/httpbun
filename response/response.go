package response

import (
	"fmt"
	"net/http"
)

type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

func New(status int, header http.Header, body []byte) Response {
	return Response{
		Status: status,
		Header: header,
		Body:   body,
	}
}

func BadRequest(message string, vars ...any) Response {
	return New(http.StatusBadRequest, nil, []byte(fmt.Sprintf(message, vars...)))
}
