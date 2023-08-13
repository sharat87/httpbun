package bun

import (
	"github.com/sharat87/httpbun/mux"
	"github.com/sharat87/httpbun/server/spec"
	"testing"
)

var BunHandler mux.Mux

func TestMain(m *testing.M) {
	BunHandler = MakeBunHandler(spec.Spec{})
	m.Run()
}
