package api_tests

import (
	"github.com/sharat87/httpbun/server"
	tu "github.com/sharat87/httpbun/test_utils"
	"testing"
)

func TestMain(m *testing.M) {
	defer server.StartNew(server.Config{
		BindTarget: tu.BindTarget,
	}).CloseAndWait()
	m.Run()
}
