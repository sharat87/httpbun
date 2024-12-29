package exchange

import (
	"io"
	"net/http"
)

func NewForTest(req http.Request, fields map[string]string) Exchange {
	return Exchange{
		Request:    &req,
		Fields:     fields,
		cappedBody: io.LimitReader(req.Body, 10000),
	}
}
