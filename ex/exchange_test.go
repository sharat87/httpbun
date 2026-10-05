package ex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowedLocationHeader(t *testing.T) {
	tests := []struct {
		name     string
		location string
		allowed  bool
	}{
		{name: "relative path", location: "/anything", allowed: true},
		{name: "relative dot path", location: "../anything", allowed: true},
		{name: "approved https domain", location: "https://example.com/path", allowed: true},
		{name: "approved http domain with port", location: "http://httpbun.com:8080/path", allowed: true},
		{name: "unknown domain", location: "https://target-url/path", allowed: false},
		{name: "non-http scheme", location: "javascript:alert(1)", allowed: false},
		{name: "scheme-relative", location: "//evil.example", allowed: false},
		{name: "backslash bypass", location: "/\\evil.example", allowed: false},
		{name: "encoded backslash bypass", location: "/%5Cevil.example", allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.allowed, isAllowedLocationHeader(tt.location, nil), "location: %q", tt.location)
		})
	}
}

func TestIsAllowedLocationHeader_CustomDomains(t *testing.T) {
	domains := []string{"custom.example"}
	assert.False(t, isAllowedLocationHeader("https://example.com/path", domains), "default domain should be disallowed when custom domains are set")
	assert.True(t, isAllowedLocationHeader("https://custom.example/path", domains), "configured domain should be allowed")
}

func TestIsAllowedLocationHeader_WildcardSubdomains(t *testing.T) {
	domains := []string{"*.github.io"}
	assert.True(t, isAllowedLocationHeader("https://docs.github.io/path", domains), "wildcard subdomain should be allowed")
	assert.True(t, isAllowedLocationHeader("https://a.b.github.io/path", domains), "nested wildcard subdomain should be allowed")
	assert.False(t, isAllowedLocationHeader("https://github.io/path", domains), "bare domain should be disallowed for wildcard-only entry")
}

func TestIsAllowedLocationHeader_NoDomains(t *testing.T) {
	assert.False(t, isAllowedLocationHeader("https://example.com/path", []string{}), "no absolute URLs should be allowed with an empty list")
	assert.True(t, isAllowedLocationHeader("/anything", []string{}), "relative paths should be allowed with an empty list")
}
