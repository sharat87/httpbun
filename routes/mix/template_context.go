package mix

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"text/template"
)

var templateFuncMap = template.FuncMap{
	"seq": tplFuncSeq,
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

type SeqItem struct {
	N       int
	IsFirst bool
	IsLast  bool
}

// Maximum number of items seq can produce, so a template can't exhaust server memory.
const maxSeqLength = 10000

func tplFuncSeq(args ...int) ([]SeqItem, error) {
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
	default:
		return nil, fmt.Errorf("seq takes one, two or three arguments, got %d", len(args))
	}
	if delta == 0 {
		return nil, errors.New("seq step can't be zero")
	}
	if (start > end && delta > 0) || (start < end && delta < 0) {
		delta = -delta
	}
	var seq []SeqItem
	for i := start; (delta > 0 && i < end) || (delta < 0 && i > end); i += delta {
		if len(seq) == maxSeqLength {
			return nil, fmt.Errorf("seq can't produce more than %d items", maxSeqLength)
		}
		seq = append(seq, SeqItem{N: i})
	}
	if len(seq) > 0 {
		seq[0].IsFirst = true
		seq[len(seq)-1].IsLast = true
	}
	return seq, nil
}
