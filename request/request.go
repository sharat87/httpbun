package request

import (
	"io"
	"net/http"
)

type Request struct {
	http.Request
	Fields     map[string]string
	CappedBody io.Reader
}

func (req Request) Field(name string) string {
	return req.Fields[name]
}
