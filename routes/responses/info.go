package responses

import (
	"encoding/json"
	"fmt"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
)

type Info struct {
	Method  string         `json:"method"`
	Args    map[string]any `json:"args"`
	Headers map[string]any `json:"headers"`
	Origin  string         `json:"origin"`
	Url     string         `json:"url"`
	Form    map[string]any `json:"form"`
	Data    any            `json:"data"` // string or []byte
	Json    *any           `json:"json"`
	Files   map[string]any `json:"files"`
}

func InfoJSON(ex *exchange.Exchange) response.Response {
	args := make(map[string]any)
	for name, values := range ex.Request.URL.Query() {
		if len(values) > 1 {
			args[name] = values
		} else {
			args[name] = values[0]
		}
	}

	result := Info{
		Method:  ex.Request.Method,
		Args:    args,
		Headers: ex.ExposableHeadersMap(),
		Origin:  ex.FindIncomingIPAddress(),
		Url:     ex.FullUrl(),
	}

	contentTypeHeaderValue := ex.HeaderValueLast(c.ContentType)
	if contentTypeHeaderValue == "" {
		contentTypeHeaderValue = "text/plain"
	}
	contentType, params, err := mime.ParseMediaType(contentTypeHeaderValue)
	if err != nil {
		return response.BadRequest("Error parsing content type %q %v.", ex.HeaderValueLast(c.ContentType), err)
	}

	form := make(map[string]any)
	var jsonData *any
	files := make(map[string]any)
	var data any // string or []byte

	if contentType == "application/x-www-form-urlencoded" {
		body := ex.BodyString()
		if parsed, err := url.ParseQuery(body); err != nil {
			data = body
		} else {
			for name, values := range parsed {
				if len(values) > 1 {
					form[name] = values
				} else {
					form[name] = values[0]
				}
			}
		}

	} else if contentType == c.ApplicationJSON {
		body := ex.BodyString()
		var result any
		if json.Unmarshal([]byte(body), &result) == nil {
			jsonData = &result
		}
		data = body

	} else if contentType == "multipart/form-data" {
		// This might work for `multipart/mixed` as well. Confirm.
		reader := multipart.NewReader(ex.Request.Body, params["boundary"])
		allFileData, err := reader.ReadForm(32 << 20)
		if err != nil {
			return response.BadRequest("Error reading multipart form data: %v", err)
		}

		for name, fileHeaders := range allFileData.File {
			fileHeader := fileHeaders[0]
			var content any
			if f, err := fileHeader.Open(); err != nil {
				fmt.Println("Error opening fileHeader", err)
			} else if content, err = io.ReadAll(f); err != nil {
				fmt.Println("Error reading from fileHeader", err)
			} else {
				if utf8.Valid(content.([]byte)) {
					content = string(content.([]byte))
				}
				headers := map[string]string{}
				for name, values := range fileHeader.Header {
					headers[name] = strings.Join(values, ",")
				}
				files[name] = map[string]any{
					"filename": fileHeader.Filename,
					"size":     fileHeader.Size,
					"headers":  headers,
					"content":  content,
				}
			}
		}

		for name, valueInfo := range allFileData.Value {
			form[name] = valueInfo[0]
		}

	} else {
		data = ex.BodyBytes()
		if utf8.Valid(data.([]byte)) {
			data = string(data.([]byte))
		}

	}

	if data == nil {
		data = ""
	}

	result.Form = form
	result.Data = data
	result.Json = jsonData
	result.Files = files

	return response.JSON(http.StatusOK, nil, result)
}
