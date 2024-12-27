package response

import (
	"fmt"
	"github.com/sharat87/httpbun/util"
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

func JSON(status int, header http.Header, body any) Response {
	if header == nil {
		header = http.Header{}
	}
	header.Set("Content-Type", "application/json")

	return Response{
		Status: status,
		Header: header,
		Body:   util.ToJsonMust(body),
	}
}

func BadRequest(message string, vars ...any) Response {
	return Response{
		Status: http.StatusBadRequest,
		Body:   fmt.Sprintf(message, vars...),
	}
}
