package util

import (
	"crypto/md5"
	crypto_rand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func WriteJson(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, ToJsonMust(data))
}

func ToJsonMust(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}

func Md5sum(text string) string {
	// Source: <https://stackoverflow.com/a/25286918/151048>.
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func RandomBytes(n int) []byte {
	b := make([]byte, n)

	if _, err := crypto_rand.Read(b); err != nil {
		fmt.Println("Error: ", err)
		return []byte{}
	}

	return b[:]
}

func RandomString() string {
	return hex.EncodeToString(RandomBytes(16))
}

func Flush(w http.ResponseWriter) bool {
	f, ok := w.(http.Flusher)
	if ok {
		f.Flush()
	}
	return ok
}

func ParseHeaderValueCsv(content string) []map[string]string {
	data := []map[string]string{}
	if content == "" {
		return data
	}

	runes := []rune(content)
	length := len(runes)
	state := "key-pre"
	key := []rune{}
	val := []rune{}
	isValueJustStarted := false
	inQuotes := false

	currentMap := make(map[string]string)

	for pos := 0; pos < length; pos++ {
		ch := runes[pos]

		if inQuotes {
			if ch == '"' {
				inQuotes = false
			} else if state == "value" {
				val = append(val, ch)
			}

		} else if ch == '=' {
			state = "value"
			isValueJustStarted = true

		} else if ch == ';' || ch == ',' {
			state = "key-pre"
			currentMap[strings.ToLower(string(key))] = string(val)
			key = []rune{}
			val = []rune{}

			if ch == ',' {
				data = append(data, currentMap)
				currentMap = make(map[string]string)
			}

		} else if state == "key-pre" {
			if ch != ' ' {
				// Whitespace just before a key is ignored.
				state = "key"
				key = append(key, ch)
			}

		} else if state == "key" {
			key = append(key, ch)

		} else if state == "value" {
			if isValueJustStarted && ch == '"' {
				inQuotes = true
			} else {
				val = append(val, ch)
			}
			isValueJustStarted = false

		}

	}

	if len(key) > 0 {
		currentMap[strings.ToLower(string(key))] = string(val)
		data = append(data, currentMap)
	}

	return data
}
