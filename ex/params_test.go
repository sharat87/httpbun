package ex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIntInRange(t *testing.T) {
	value, err := parseIntInRange("count", "5", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 5, value)

	for _, raw := range []string{"", "abc", "0", "11", "1.5", "99999999999999999999"} {
		_, err := parseIntInRange("count", raw, 1, 10)
		assert.EqualError(t, err, `count must be an integer between 1 and 10, got "`+raw+`"`)
	}
}
