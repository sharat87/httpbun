package util

import (
	"bytes"
	"crypto/md5"
	cryptoRand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
)

func ToJsonMust(data any) []byte {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(data)
	if err != nil {
		log.Fatal(err)
	}
	return append(bytes.TrimSpace(buffer.Bytes()), '\n')
}

func Md5sum(text string) string {
	// Source: <https://stackoverflow.com/a/25286918/151048>.
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func RandomBytes(n int) []byte {
	b := make([]byte, n)

	if _, err := cryptoRand.Read(b); err != nil {
		fmt.Println("Error: ", err)
		return []byte{}
	}

	return b[:]
}

func RandomString() string {
	return hex.EncodeToString(RandomBytes(16))
}

func CommitHashShorten(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

func ComputeFgForBg(c string) string {
	rgb, err := strconv.ParseInt(c[1:], 16, 32)
	if err != nil {
		fmt.Println("Error parsing color:", err)
		return "black"
	}
	r := (rgb >> 16) & 0xff
	g := (rgb >> 8) & 0xff
	b := rgb & 0xff

	luma := 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b) // per ITU-R BT.709

	if luma > 100 {
		return "#222e"
	} else {
		return "#eeee"
	}
}

// CheckStatusCode checks that a status code can be used for a response. 1xx codes are informational, and can't be the
// final status of a response, so they aren't allowed.
func CheckStatusCode(code int) error {
	if code < 200 || code > 599 {
		return fmt.Errorf("invalid status code %d, must be between 200 and 599", code)
	}
	return nil
}

// ParseStatusCode parses a status code that can be used for a response, as checked by CheckStatusCode.
func ParseStatusCode(raw string) (int, error) {
	code, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid status code %q", raw)
	}
	return code, CheckStatusCode(code)
}
