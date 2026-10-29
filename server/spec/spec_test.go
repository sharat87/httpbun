package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyEnv(t *testing.T) {
	s := assert.New(t)
	var spec Spec
	applyEnv(&spec, []string{
		"HTTPBUN_TLS_CERT=/cert.pem",
		"HTTPBUN_TLS_KEY=/key.pem",
		"HTTPBUN_ALLOWED_REDIRECT_DOMAINS=alpha.example,\nbeta.example  *.gamma.example",
		"HTTPBUN_INFO_REGION=eu",
		"SOME_SECRET_TOKEN=hunter2",
	})
	s.Equal("/cert.pem", spec.TLSCertFile)
	s.Equal("/key.pem", spec.TLSKeyFile)
	s.Equal([]string{"alpha.example", "beta.example", "*.gamma.example"}, spec.AllowedRedirectDomains)
	s.Equal(map[string]string{"HTTPBUN_INFO_REGION": "eu"}, spec.InfoEnv)
}

func TestApplyEnvDefaults(t *testing.T) {
	s := assert.New(t)
	var spec Spec
	applyEnv(&spec, []string{"PATH=/bin"})
	s.Nil(spec.AllowedRedirectDomains, "nil means the default domains")
	s.Empty(spec.InfoEnv)
}

func TestApplyEnvEmptyRedirectDomains(t *testing.T) {
	var spec Spec
	applyEnv(&spec, []string{"HTTPBUN_ALLOWED_REDIRECT_DOMAINS="})
	assert.Equal(t, []string{}, spec.AllowedRedirectDomains, "set but empty allows no domains")
}

func TestApplyEnvProxyOptions(t *testing.T) {
	s := assert.New(t)
	var spec Spec
	applyEnv(&spec, []string{
		"HTTPBUN_CLIENT_IP_HEADER=X-Real-Ip",
		"HTTPBUN_SCHEME_HEADER=X-Forwarded-Proto",
		"HTTPBUN_HIDE_HEADERS=X-Forwarded-*, X-Request-Start",
		"HTTPBUN_ANY_HOSTS=any.example.com",
		"HTTPBUN_HSTS=true",
	})
	s.Equal("X-Real-Ip", spec.ClientIPHeaderName())
	s.Equal("X-Forwarded-Proto", spec.SchemeHeaderName())
	s.Equal([]string{"X-Forwarded-*", "X-Request-Start"}, spec.HideHeaders)
	s.Equal([]string{"any.example.com"}, spec.AnyHosts)
	s.True(spec.HSTS)
}

func TestProxyOptionDefaults(t *testing.T) {
	s := assert.New(t)
	var spec Spec
	s.Equal(DefaultClientIPHeader, spec.ClientIPHeaderName())
	s.Equal(DefaultSchemeHeader, spec.SchemeHeaderName())
	s.False(spec.HSTS)
	s.False(spec.IsHiddenHeader("X-Real-Ip"), "nothing beyond X-Httpbun-* is hidden by default")
	s.True(spec.IsHiddenHeader("X-Httpbun-Anything"))
	s.False(spec.IsAnyHost("any.example.com"))
}

func TestIsHiddenHeader(t *testing.T) {
	s := assert.New(t)
	spec := Spec{ClientIPHeader: "X-Real-Ip", SchemeHeader: "X-Forwarded-Proto", HideHeaders: []string{"X-Railway-*", "via"}}
	s.True(spec.IsHiddenHeader("X-Real-Ip"), "the client IP header is hidden")
	s.True(spec.IsHiddenHeader("x-forwarded-proto"), "the scheme header is hidden, case insensitively")
	s.True(spec.IsHiddenHeader("X-Railway-Edge"))
	s.True(spec.IsHiddenHeader("Via"))
	s.False(spec.IsHiddenHeader("X-Forwarded-For"))
	s.False(spec.IsHiddenHeader("Accept"))
}

func TestIsAnyHost(t *testing.T) {
	s := assert.New(t)
	spec := Spec{AnyHosts: []string{"any.example.com"}}
	s.True(spec.IsAnyHost("any.example.com"))
	s.True(spec.IsAnyHost("ANY.example.com:8080"))
	s.False(spec.IsAnyHost("example.com"))
	s.False(spec.IsAnyHost("other.any.example.com"))
}

func TestApplyEnvLimits(t *testing.T) {
	s := assert.New(t)
	var spec Spec
	s.Equal(DefaultMaxConcurrentRequests, spec.MaxConcurrentRequestsLimit())
	s.Equal(DefaultMaxBodySize, spec.MaxBodySizeLimit())

	s.NoError(applyEnv(&spec, []string{
		"HTTPBUN_MAX_CONCURRENT_REQUESTS=8",
		"HTTPBUN_MAX_BODY_SIZE=1024",
	}))
	s.Equal(8, spec.MaxConcurrentRequestsLimit())
	s.Equal(int64(1024), spec.MaxBodySizeLimit())
}

func TestApplyEnvInvalidLimits(t *testing.T) {
	for _, e := range []string{
		"HTTPBUN_MAX_CONCURRENT_REQUESTS=0",
		"HTTPBUN_MAX_CONCURRENT_REQUESTS=lots",
		"HTTPBUN_MAX_BODY_SIZE=-1",
	} {
		var spec Spec
		assert.Error(t, applyEnv(&spec, []string{e}), e)
	}
}
