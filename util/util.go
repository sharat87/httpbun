package util

import (
	"bytes"
	"crypto/md5"
	cryptoRand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
)

func ToJson(data any) []byte {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(data)
	if err != nil {
		log.Printf("Error encoding JSON: %v", err)
		return []byte(err.Error())
	}
	return bytes.TrimSpace(buffer.Bytes())
}

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
