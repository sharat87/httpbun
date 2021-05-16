package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOne(t *testing.T) {
	results := ParseHeaderValueCsv("for=12.34.56.78")
	assert.Equal(
		t,
		[]map[string]string{
			map[string]string{
				"for": "12.34.56.78",
			},
		},
		results,
	)
}

func TestTwo(t *testing.T) {
	results := ParseHeaderValueCsv("for=12.34.56.78;host=example.com;proto=https, for=23.45.67.89")
	assert.Equal(
		t,
		[]map[string]string{
			map[string]string{
				"for":   "12.34.56.78",
				"host":  "example.com",
				"proto": "https",
			},
			map[string]string{
				"for": "23.45.67.89",
			},
		},
		results,
	)
}

func TestThree(t *testing.T) {
	results := ParseHeaderValueCsv("for=12.34.56.78, for=23.45.67.89;secret=egah2CGj55fSJFs, for=10.1.2.3")
	assert.Equal(
		t,
		[]map[string]string{
			map[string]string{
				"for": "12.34.56.78",
			},
			map[string]string{
				"for":    "23.45.67.89",
				"secret": "egah2CGj55fSJFs",
			},
			map[string]string{
				"for": "10.1.2.3",
			},
		},
		results,
	)
}

func TestFour(t *testing.T) {
	results := ParseHeaderValueCsv(`for="_gazonk"`)
	assert.Equal(
		t,
		[]map[string]string{
			map[string]string{
				"for": "_gazonk",
			},
		},
		results,
	)
}

func TestFive(t *testing.T) {
	results := ParseHeaderValueCsv(`For="[2001:db8:cafe::17]:4711"`)
	assert.Equal(
		t,
		[]map[string]string{
			map[string]string{
				"for": "[2001:db8:cafe::17]:4711",
			},
		},
		results,
	)
}

func TestSix(t *testing.T) {
	results := ParseHeaderValueCsv(`for=192.0.2.60;proto=http;by=203.0.113.43`)
	assert.Equal(
		t,
		[]map[string]string{
			map[string]string{
				"for":   "192.0.2.60",
				"proto": "http",
				"by":    "203.0.113.43",
			},
		},
		results,
	)
}

func TestSeven(t *testing.T) {
	results := ParseHeaderValueCsv(`for=192.0.2.43, for=198.51.100.17`)
	assert.Equal(
		t,
		[]map[string]string{
			map[string]string{
				"for": "192.0.2.43",
			},
			map[string]string{
				"for": "198.51.100.17",
			},
		},
		results,
	)
}

func TestSemicolonInValue(t *testing.T) {
	results := ParseHeaderValueCsv(`for=1.2.3.4;secret="abc;def"`)
	assert.Equal(
		t,
		[]map[string]string{
			map[string]string{
				"for":    "1.2.3.4",
				"secret": "abc;def",
			},
		},
		results,
	)
}

func TestCommaInValue(t *testing.T) {
	results := ParseHeaderValueCsv(`for=1.2.3.4;secret="abc,def"`)
	assert.Equal(
		t,
		[]map[string]string{
			map[string]string{
				"for":    "1.2.3.4",
				"secret": "abc,def",
			},
		},
		results,
	)
}

func TestEqualsInValue(t *testing.T) {
	results := ParseHeaderValueCsv(`for=1.2.3.4;secret="abc=def"`)
	assert.Equal(
		t,
		[]map[string]string{
			map[string]string{
				"for":    "1.2.3.4",
				"secret": "abc=def",
			},
		},
		results,
	)
}
