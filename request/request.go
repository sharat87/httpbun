package request

import (
	"net/http"
	"io"
)

type Request struct {
	http.Request
	Fields     map[string]string
	CappedBody io.Reader
}

func (req Request) Field(name string) string {
	return req.Fields[name]
}
