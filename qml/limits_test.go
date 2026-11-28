package qml

import (
	"strings"
	"testing"
)

func TestParseQMLRejectsDeepNesting(t *testing.T) {
	src := "Window {" + strings.Repeat("Rectangle {", maxQMLNesting+1) +
		strings.Repeat("}", maxQMLNesting+1) + "}"
	if _, err := ParseQML(src); err == nil || !strings.Contains(err.Error(), "nesting") {
		t.Fatalf("deep nesting: err = %v, want a nesting error", err)
	}
}

func TestParseQMLAcceptsReasonableNesting(t *testing.T) {
	src := "Window {" + strings.Repeat("Rectangle {", 8) + strings.Repeat("}", 8) + "}"
	if _, err := ParseQML(src); err != nil {
		t.Fatalf("8 levels should parse: %v", err)
	}
}

func TestParseQMLRejectsOversizedSource(t *testing.T) {
	src := "Window {" + strings.Repeat(" ", maxQMLBytes+1) + "}"
	if _, err := ParseQML(src); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("oversized source: err = %v, want a size error", err)
	}
}
