package api_tests

import (
	"github.com/sharat87/httpbun/server"
	"github.com/sharat87/httpbun/server/spec"
	tu "github.com/sharat87/httpbun/test_utils"
	"testing"
)

func TestMain(m *testing.M) {
	defer server.StartNew(spec.Spec{
		BindTarget: tu.BindTarget,
	}).CloseAndWait()
	m.Run()
}
