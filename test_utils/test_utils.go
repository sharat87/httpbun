package test_utils

import (
	"encoding/json"
)

type R struct {
	Method  string
	Path    string
	Body    string
	Headers map[string][]string
}

func ParseJson(raw []byte) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		panic(err)
	}
	return data
}
