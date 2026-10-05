package run

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dop251/goja"

	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
)

const restPathPattern = `/(?P<encoded>[-\w]+=*)(?P<extraPath>.*)`

var RouteList = []ex.Route{
	ex.NewRoute(`/runner(`+restPathPattern+")?", handleRunner),
	ex.NewRoute(`/run`+restPathPattern, handleRunJS),
}

func handleRunner(ex *ex.Exchange) response.Response {
	return assets.Render("runner.html", *ex, nil)
}

// makeReadOnlyRequestJS builds the `R` object from JSON, as plain objects that throw on any attempt to change them.
// Changing `R` can't affect anything, so it's better to fail early than to silently ignore it. Scripts don't run in
// strict mode, where a frozen object would silently ignore changes, hence the Proxy.
const makeReadOnlyRequestJS = `(json) => {
	const readOnly = (obj, label) => new Proxy(obj, {
		set(_, name) {
			throw new TypeError(label + " is read-only, can't set " + String(name) +
				". To set response headers, return them in the headers field.")
		},
		deleteProperty(_, name) {
			throw new TypeError(label + " is read-only, can't delete " + String(name))
		},
		defineProperty(_, name) {
			throw new TypeError(label + " is read-only, can't define " + String(name))
		},
	})
	const r = JSON.parse(json)
	r.headers = readOnly(r.headers, "R.headers")
	return readOnly(r, "R")
}`

func handleRunJS(ex *ex.Exchange) response.Response {
	src, err := base64.URLEncoding.DecodeString(ex.Field("encoded"))
	if err != nil {
		return response.BadRequest("Invalid encoded data: %s", err.Error())
	}

	rt := goja.New()
	time.AfterFunc(100*time.Millisecond, func() {
		rt.Interrupt("halt")
	})

	rawFn, err := rt.RunString("R => {\n" + string(src) + "\n}")
	if err != nil {
		return response.BadRequest("Evaluation error: %s", err.Error())
	}

	fn, ok := goja.AssertFunction(rawFn)
	if !ok {
		return response.BadRequest("Unable to load JS")
	}

	// A plain object, so it works with normal JS syntax, like `R.headers["user-agent"]`. Names are lowercase, and
	// repeated headers are joined with a comma, like HTTP allows.
	requestHeaders := map[string]any{}
	for name, values := range ex.Request.Header {
		requestHeaders[strings.ToLower(name)] = strings.Join(values, ", ")
	}

	rParam := map[string]any{
		"method":    ex.Request.Method,
		"headers":   requestHeaders,
		"extraPath": ex.Field("extraPath"),
	}

	rParamJSON, err := json.Marshal(rParam)
	if err != nil {
		return response.BadRequest("Unable to prepare request: %s", err.Error())
	}
	rawMakeR, err := rt.RunString(makeReadOnlyRequestJS)
	if err != nil {
		return response.BadRequest("Unable to prepare request: %s", err.Error())
	}
	makeR, _ := goja.AssertFunction(rawMakeR)
	r, err := makeR(goja.Undefined(), rt.ToValue(string(rParamJSON)))
	if err != nil {
		return response.BadRequest("Unable to prepare request: %s", err.Error())
	}

	rawResult, err := fn(goja.Undefined(), r)
	if err != nil {
		return response.BadRequest("Evaluation error: %s", err.Error())
	}

	result, ok := rawResult.Export().(map[string]any)
	if !ok {
		return response.BadRequest("Evaluation error: script must return an object, like `return {body: \"hello\"}`")
	}

	status := 0
	if statusRaw, haveStatus := result["status"]; haveStatus {
		if statusInt, ok := statusRaw.(int64); ok {
			status = int(statusInt)
		} else {
			return response.BadRequest("Evaluation error: status is not an integer")
		}
	}
	if status == 0 {
		status = 200
	}
	if status < 200 || status > 599 {
		// 1xx codes are informational, and can't be sent as the final status of a response.
		return response.BadRequest("Evaluation error: status must be between 200 and 599")
	}

	var headers http.Header
	if headersRaw, haveHeaders := result["headers"]; haveHeaders {
		switch headersTyped := headersRaw.(type) {
		case map[string]any:
			headers = http.Header{}
			for k, v := range headersTyped {
				if vString, isString := v.(string); isString {
					headers.Add(k, vString)
				} else if vList, isList := v.([]any); isList {
					for _, item := range vList {
						vString, isString := item.(string)
						if !isString {
							return response.BadRequest("Invalid header value type for key: %s", k)
						}
						headers.Add(k, vString)
					}
				} else {
					return response.BadRequest("Invalid header value type for key: %s", k)
				}
			}
		case nil:
			headers = nil
		default:
			return response.BadRequest("Invalid headers value: %v", headersRaw)
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
				return response.BadRequest("Body JSON stringify error: %s", err.Error())
			}
		}
	}

	return response.New(status, headers, body)
}
