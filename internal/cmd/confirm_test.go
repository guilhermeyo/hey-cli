package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmFromReader_Yes(t *testing.T) {
	r := strings.NewReader("y\n")
	var buf bytes.Buffer
	if !confirmFromReader("Do it?", r, &buf) {
		t.Error("expected true for 'y' input")
	}
	if !strings.Contains(buf.String(), "Do it?") {
		t.Error("expected prompt in output")
	}
}

func TestConfirmFromReader_No(t *testing.T) {
	r := strings.NewReader("n\n")
	var buf bytes.Buffer
	if confirmFromReader("Do it?", r, &buf) {
		t.Error("expected false for 'n' input")
	}
}

func TestConfirmFromReader_EmptyDefault(t *testing.T) {
	r := strings.NewReader("\n")
	var buf bytes.Buffer
	if confirmFromReader("Do it?", r, &buf) {
		t.Error("expected false for empty input (default N)")
	}
}

func TestConfirmFromReader_YesUpperCase(t *testing.T) {
	r := strings.NewReader("YES\n")
	var buf bytes.Buffer
	if !confirmFromReader("Do it?", r, &buf) {
		t.Error("expected true for 'YES' input")
	}
}
