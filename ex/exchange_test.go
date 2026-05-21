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

func TestIsAllowedLocationHeader_EnvOverride(t *testing.T) {
	t.Setenv(allowedRedirectDomainsEnvVar, "custom.example")

	if isAllowedLocationHeader("https://example.com/path") {
		t.Fatal("expected default domain to be disallowed when env override is set")
	}

	if !isAllowedLocationHeader("https://custom.example/path") {
		t.Fatal("expected configured domain to be allowed")
	}
}

func TestIsAllowedLocationHeader_EnvSplitPatterns(t *testing.T) {
	t.Setenv(allowedRedirectDomainsEnvVar, "alpha.example,\nbeta.example  gamma.example")

	tests := []string{
		"https://alpha.example/path",
		"https://beta.example/path",
		"https://gamma.example/path",
	}

	for _, location := range tests {
		if !isAllowedLocationHeader(location) {
			t.Fatalf("expected %q to be allowed", location)
		}
	}
}

func TestIsAllowedLocationHeader_WildcardSubdomains(t *testing.T) {
	t.Setenv(allowedRedirectDomainsEnvVar, "*.github.io")

	if !isAllowedLocationHeader("https://docs.github.io/path") {
		t.Fatal("expected wildcard subdomain to be allowed")
	}

	if !isAllowedLocationHeader("https://a.b.github.io/path") {
		t.Fatal("expected nested wildcard subdomain to be allowed")
	}

	if isAllowedLocationHeader("https://github.io/path") {
		t.Fatal("expected bare domain to be disallowed for wildcard-only entry")
	}
}
