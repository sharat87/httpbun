package run

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dop251/goja"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"net/http"
	"time"
)

var Routes = map[string]exchange.HandlerFn2{
	`/run/(?P<encoded>[-=%\w]+)(?P<extraPath>.*)`: handleRunJS,
}

func handleRunJS(ex *exchange.Exchange) response.Response {
	src, err := base64.URLEncoding.DecodeString(ex.Field("encoded"))
	if err != nil {
		return response.BadRequest("Invalid encoded data: " + err.Error())
	}

	rt := goja.New()
	time.AfterFunc(200*time.Millisecond, func() {
		rt.Interrupt("halt")
	})

	rawFn, err := rt.RunString("R => {\n" + string(src) + "\n}")
	if err != nil {
		return response.BadRequest("Evaluation error: " + err.Error())
	}

	fn, ok := goja.AssertFunction(rawFn)
	if !ok {
		return response.BadRequest("Unable to load JS: " + err.Error())
	}

	rParam := map[string]any{
		"method":    ex.Request.Method,
		"headers":   ex.Request.Header,
		"extraPath": ex.Field("extraPath"),
	}

	rawResult, err := fn(goja.Undefined(), rt.ToValue(rParam))
	if err != nil {
		return response.BadRequest("Evaluation error: " + err.Error())
	}

	result := rawResult.Export().(map[string]any)

	status := 0
	if statusRaw, haveStatus := result["status"]; haveStatus {
		if statusInt, ok := statusRaw.(int64); ok {
			status = int(statusInt)
		} else {
			return response.BadRequest("Evaluation error: status is not an integer")
		}
	}
	if status < 0 {
		return response.BadRequest("Evaluation error: status is negative")
	}
	if status == 0 {
		status = 200
	}

	var headers http.Header
	if headersRaw, haveHeaders := result["headers"]; haveHeaders {
		switch headersTyped := headersRaw.(type) {
		case map[string]any:
			headers = http.Header{}
			for k, v := range headersTyped {
				if vString, isString := v.(string); isString {
					headers.Add(k, vString)
				} else if vList, isList := v.([]string); isList {
					for _, vString := range vList {
						headers.Add(k, vString)
					}
				} else {
					return response.BadRequest("Invalid header value type for key: " + k)
				}
			}
		case nil:
			headers = nil
		default:
			return response.BadRequest("Invalid headers value: " + fmt.Sprintf("%v", headersRaw))
		}
	}

	var body []byte
	if bodyRaw, haveBody := result["body"]; haveBody {
		switch bodyTyped := bodyRaw.(type) {
		case string:
			body = []byte(bodyTyped)
		default:
			body, err = json.Marshal(bodyTyped)
			if err != nil {
				return response.BadRequest("Body JSON stringify error: " + err.Error())
			}
		}
	}

	return response.New(status, headers, body)
}
