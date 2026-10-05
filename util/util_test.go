package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStatusCode(t *testing.T) {
	for raw, want := range map[string]int{"200": 200, " 418 ": 418, "599": 599} {
		code, err := ParseStatusCode(raw)
		assert.NoError(t, err, raw)
		assert.Equal(t, want, code)
	}
	for _, raw := range []string{"", "abc", "100", "199", "600", "-1"} {
		_, err := ParseStatusCode(raw)
		assert.Error(t, err, raw)
	}
}
