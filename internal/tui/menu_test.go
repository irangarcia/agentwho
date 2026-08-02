package tui

import (
	"strings"
	"testing"
)

func TestMoveSelectionWraps(t *testing.T) {
	tests := []struct {
		name                  string
		current, count, delta int
		want                  int
	}{
		{"down", 0, 2, 1, 1},
		{"down wraps", 1, 2, 1, 0},
		{"up", 1, 2, -1, 0},
		{"up wraps", 0, 2, -1, 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := moveSelection(test.current, test.count, test.delta); got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestReadMenuKey(t *testing.T) {
	tests := []struct {
		input string
		want  menuKey
	}{
		{"\x1b[A", menuUp},
		{"\x1b[B", menuDown},
		{"k", menuUp},
		{"j", menuDown},
		{"\r", menuChoose},
		{"\x03", menuCancel},
	}
	for _, test := range tests {
		got, err := readMenuKey(strings.NewReader(test.input))
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Errorf("input %q: got %v, want %v", test.input, got, test.want)
		}
	}
}
