package mix

import (
	"bytes"
	"encoding/json"
	"log"
	"text/template"
)

var templateFuncMap = template.FuncMap{
	"seq":    tplFuncSeq,
	"repeat": tplFuncSeq,
	"toJSON": func(v any) string {
		buffer := &bytes.Buffer{}
		encoder := json.NewEncoder(buffer)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(v)
		if err != nil {
			log.Printf("Error encoding JSON: %v", err)
			return err.Error()
		}
		return string(bytes.TrimSpace(buffer.Bytes()))
	},
}

func tplFuncSeq(args ...int) []map[string]any {
	var start, end, delta int
	switch len(args) {
	case 1:
		start = 0
		end = args[0]
		delta = 1
	case 2:
		start = args[0]
		end = args[1]
		delta = 1
	case 3:
		start = args[0]
		end = args[1]
		delta = args[2]
	}
	if (start > end && delta > 0) || (start < end && delta < 0) {
		delta = -delta
	}
	var seq []map[string]any
	for i := start; i != end; i += delta {
		seq = append(seq, map[string]any{
			"N":       i,
			"IsFirst": false,
			"IsLast":  false,
		})
	}
	if len(seq) > 0 {
		seq[0]["IsFirst"] = true
		seq[len(seq)-1]["IsLast"] = true
	}
	return seq
}
