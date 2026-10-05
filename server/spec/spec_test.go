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
