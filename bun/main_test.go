package bun

import (
	"github.com/sharat87/httpbun/mux"
	"testing"
)

var BunHandler mux.Mux

func TestMain(m *testing.M) {
	BunHandler = MakeBunHandler("", "", "")
	m.Run()
}
