package run

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dop251/goja"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	lua "github.com/yuin/gopher-lua"
	"net/http"
	"time"
)

var Routes = map[string]exchange.HandlerFn2{
	"/run/lua/(?P<encoded>.*)":                 handleRunLua,
	"/run/(?P<encoded>[^/]+)(?P<extraPath>.*)": handleRunJS,
}

type EvalResult struct {
	status  int
	headers http.Header
	boty    string
}

func handleRunLua(ex *exchange.Exchange) response.Response {
	src, err := base64.StdEncoding.DecodeString(ex.Field("encoded"))
	if err != nil {
		return response.BadRequest("Invalid encoded data: " + err.Error())
	}

	// https://github.com/yuin/gopher-lua?tab=readme-ov-file#opening-a-subset-of-builtin-modules
	state := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer state.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	state.SetContext(ctx)

	if err := state.DoString(string(src)); err != nil {
		return response.BadRequest("Evaluation error: " + err.Error())
	}

	result, isTable := state.Get(-1).(*lua.LTable)
	if !isTable {
		return response.BadRequest("Return value is not a Lua table")
	}

	status := GetStatus(state, result, "status")

	headers, err := GetHeaders(state, result)
	if err != nil {
		return response.BadRequest("Error reading headers: " + err.Error())
	}

	body, err := GetBody(state, result)
	if err != nil {
		return response.BadRequest("Error reading body: " + err.Error())
	}

	return response.New(status, headers, body)
}

func GetStatus(state *lua.LState, tbl *lua.LTable, key string) int {
	field := state.GetField(tbl, key)

	if field.Type() != lua.LTNil {
		if field.Type() == lua.LTNumber {
			return int(field.(lua.LNumber))
		}
	}

	return 200
}

func GetHeaders(state *lua.LState, tbl *lua.LTable) (http.Header, error) {
	field := state.GetField(tbl, "headers")

	if field.Type() == lua.LTNil {
		return nil, nil
	}

	headers := make(http.Header)
	if field.Type() == lua.LTTable {
		var entries [][]lua.LValue
		field.(*lua.LTable).ForEach(func(name, value lua.LValue) {
			entries = append(entries, []lua.LValue{name, value})
		})

		for _, entry := range entries {
			name := entry[0]
			value := entry[1]

			if name.Type() != lua.LTString {
				return nil, fmt.Errorf("headers key %v is not a string", name)
			}

			if value.Type() == lua.LTString {
				headers.Add(name.String(), value.String())
			} else if value.Type() == lua.LTTable {
				value.(*lua.LTable).ForEach(func(_, value lua.LValue) {
					headers.Add(name.String(), value.String())
				})
			} else {
				return nil, fmt.Errorf("headers value for %v is not a lua string or table", name)
			}
		}
	}

	return headers, nil
}

func GetBody(state *lua.LState, tbl *lua.LTable) ([]byte, error) {
	field := state.GetField(tbl, "body")

	if field.Type() == lua.LTNil {
		return nil, nil
	}

	if field.Type() == lua.LTString {
		return []byte(field.String()), nil
	}

	data, err := luaValueToJSON(field)
	if err != nil {
		return nil, err
	}

	return data, nil
}

type LValueWrap struct {
	lua.LValue
	visited map[*lua.LTable]bool
}

func luaValueToJSON(value lua.LValue) ([]byte, error) {
	return json.Marshal(LValueWrap{
		LValue:  value,
		visited: make(map[*lua.LTable]bool),
	})
}

func (v LValueWrap) MarshalJSON() (data []byte, err error) {
	switch converted := v.LValue.(type) {
	case lua.LBool:
		data, err = json.Marshal(bool(converted))
	case lua.LNumber:
		data, err = json.Marshal(float64(converted))
	case *lua.LNilType:
		data = []byte("null")
	case lua.LString:
		data, err = json.Marshal(string(converted))
	case *lua.LTable:
		if v.visited[converted] {
			return nil, fmt.Errorf("JSON serialize: cycle detected")
		}
		v.visited[converted] = true

		key, value := converted.Next(lua.LNil)

		switch key.Type() {
		case lua.LTNil: // empty table
			data = []byte(`[]`)
		case lua.LTNumber:
			arr := make([]LValueWrap, 0, converted.Len())
			expectedKey := lua.LNumber(1)
			for key != lua.LNil {
				if key.Type() != lua.LTNumber {
					err = fmt.Errorf("cannot encode mixed or invalid key types")
					return
				}
				if expectedKey != key {
					err = fmt.Errorf("cannot encode sparse array")
					return
				}
				arr = append(arr, LValueWrap{value, v.visited})
				expectedKey++
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(arr)
		case lua.LTString:
			obj := make(map[string]LValueWrap)
			for key != lua.LNil {
				if key.Type() != lua.LTString {
					err = fmt.Errorf("cannot encode mixed or invalid key types")
					return
				}
				obj[key.String()] = LValueWrap{value, v.visited}
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(obj)
		default:
			err = fmt.Errorf("cannot encode mixed or invalid key types")
		}
	default:
		err = fmt.Errorf("cannot encode %s to JSON", v.LValue.Type())
	}

	return
}

func handleRunJS(ex *exchange.Exchange) response.Response {
	src, err := base64.StdEncoding.DecodeString(ex.Field("encoded"))
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
