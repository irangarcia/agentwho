package termstyle

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPaintOnlyUsesColorWhenEnabled(t *testing.T) {
	oldNoColor, hadNoColor := os.LookupEnv("NO_COLOR")
	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if hadNoColor {
			_ = os.Setenv("NO_COLOR", oldNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	})
	t.Setenv("CLICOLOR_FORCE", "")
	var output bytes.Buffer
	if got := Paint(&output, Success, "ready"); got != "ready" {
		t.Fatalf("non-terminal output contains color: %q", got)
	}

	t.Setenv("CLICOLOR_FORCE", "1")
	got := Paint(&output, Success, "ready")
	if !strings.HasPrefix(got, "\x1b[") || !strings.HasSuffix(got, "\x1b[0m") {
		t.Fatalf("forced color was not applied: %q", got)
	}
}

func TestNoColorOverridesForcedColor(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	t.Setenv("NO_COLOR", "1")
	if got := Paint(&bytes.Buffer{}, Danger, "blocked"); got != "blocked" {
		t.Fatalf("NO_COLOR was ignored: %q", got)
	}
}

func TestPaletteMatchesDocumentationAssets(t *testing.T) {
	oldNoColor, hadNoColor := os.LookupEnv("NO_COLOR")
	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if hadNoColor {
			_ = os.Setenv("NO_COLOR", oldNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	})
	t.Setenv("CLICOLOR_FORCE", "1")
	tests := []struct {
		name  string
		style Style
		code  string
	}{
		{"accent purple", Accent, "1;38;2;210;168;255"},
		{"info blue", Info, "1;38;2;121;192;255"},
		{"success green", Success, "1;38;2;86;211;100"},
		{"warning yellow", Warning, "1;38;2;227;179;65"},
		{"danger red", Danger, "1;38;2;255;123;114"},
		{"muted gray", Muted, "38;2;139;148;158"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := "\x1b[" + tt.code + "mtext\x1b[0m"
			if got := Paint(&bytes.Buffer{}, tt.style, "text"); got != want {
				t.Fatalf("palette drift: got %q, want %q", got, want)
			}
		})
	}
}
