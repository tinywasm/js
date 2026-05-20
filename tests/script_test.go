package js_test

import (
	"testing"

	"github.com/tinywasm/js"
)

func TestScript_StringReturnsContent(t *testing.T) {
	s := &js.Script{Content: "x"}
	if got := s.String(); got != "x" {
		t.Errorf("Script.String() = %q, want %q", got, "x")
	}
}

func TestScript_ZeroValue(t *testing.T) {
	s := &js.Script{}
	if got := s.String(); got != "" {
		t.Errorf("Script.String() = %q, want empty string", got)
	}
	if s.Name != "" {
		t.Errorf("Script.Name = %q, want empty string", s.Name)
	}
}
