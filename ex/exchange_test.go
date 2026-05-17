package ex

import "testing"

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
			if got := isAllowedLocationHeader(tt.location); got != tt.allowed {
				t.Fatalf("isAllowedLocationHeader(%q) = %v, want %v", tt.location, got, tt.allowed)
			}
		})
	}
}
