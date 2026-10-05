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
			if got := isAllowedLocationHeader(tt.location, nil); got != tt.allowed {
				t.Fatalf("isAllowedLocationHeader(%q) = %v, want %v", tt.location, got, tt.allowed)
			}
		})
	}
}

func TestIsAllowedLocationHeader_CustomDomains(t *testing.T) {
	domains := []string{"custom.example"}

	if isAllowedLocationHeader("https://example.com/path", domains) {
		t.Fatal("expected default domain to be disallowed when custom domains are set")
	}

	if !isAllowedLocationHeader("https://custom.example/path", domains) {
		t.Fatal("expected configured domain to be allowed")
	}
}

func TestIsAllowedLocationHeader_WildcardSubdomains(t *testing.T) {
	domains := []string{"*.github.io"}

	if !isAllowedLocationHeader("https://docs.github.io/path", domains) {
		t.Fatal("expected wildcard subdomain to be allowed")
	}

	if !isAllowedLocationHeader("https://a.b.github.io/path", domains) {
		t.Fatal("expected nested wildcard subdomain to be allowed")
	}

	if isAllowedLocationHeader("https://github.io/path", domains) {
		t.Fatal("expected bare domain to be disallowed for wildcard-only entry")
	}
}

func TestIsAllowedLocationHeader_NoDomains(t *testing.T) {
	if isAllowedLocationHeader("https://example.com/path", []string{}) {
		t.Fatal("expected no absolute URLs to be allowed with an empty list")
	}
	if !isAllowedLocationHeader("/anything", []string{}) {
		t.Fatal("expected relative paths to be allowed with an empty list")
	}
}
