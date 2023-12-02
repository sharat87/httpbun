package auth

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestComputeDigestAuthResponse(t *testing.T) {
	response := computeDigestAuthResponse(
		"Mufasa",
		"Circle Of Life",
		"dcd98b7102dd2f0e8b11d0f600bfb0c093",
		"00000001",
		"0a4f113b",
		"auth",
		"GET",
		"/dir/index.html",
	)
	assert.Equal(
		t,
		"6629fae49393a05397450978507c4ef1",
		response,
	)
}
