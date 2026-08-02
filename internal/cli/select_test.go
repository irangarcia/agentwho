package cli

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestSelectOneNumberedFallback(t *testing.T) {
	var output bytes.Buffer
	a := &app{in: strings.NewReader("2\n"), out: &output}
	got, err := a.selectOne(bufio.NewReader(a.in), "Choose safety mode", safetyModeOptions(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != "confirm" {
		t.Fatalf("got %q, want confirm", got)
	}
	for _, want := range []string{"Choose safety mode", "1. Block", "2. Confirm", "Choice [1]", "✓ Selected: Confirm"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output missing %q:\n%s", want, output.String())
		}
	}
}

func TestSelectOneFallbackAcceptsValue(t *testing.T) {
	a := &app{in: strings.NewReader("block\n"), out: &bytes.Buffer{}}
	got, err := a.selectOne(bufio.NewReader(a.in), "Choose", safetyModeOptions(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if got != "block" {
		t.Fatalf("got %q, want block", got)
	}
}
