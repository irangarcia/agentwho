package enforce

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/irangarcia/agentwho/internal/config"
)

func mismatch(mode string) Input {
	return Input{Expected: "work", Selected: "personal", Enforcement: mode, AgentDisplayName: "Claude Code", Profiles: map[string]config.Profile{"work": {Kind: "work"}, "personal": {Kind: "personal"}}}
}

func TestMismatchModes(t *testing.T) {
	tests := []struct {
		name    string
		in      Input
		input   string
		profile string
		wantErr bool
	}{
		{"same", Input{Expected: "work", Selected: "work"}, "", "work", false},
		{"block", mismatch("block"), "", "", true},
		{"confirm switch expected", func() Input { x := mismatch("confirm"); x.Interactive = true; return x }(), "1\n", "work", false},
		{"confirm continue selected", func() Input { x := mismatch("confirm"); x.Interactive = true; return x }(), "2\n", "personal", false},
		{"confirm cancel", func() Input { x := mismatch("confirm"); x.Interactive = true; return x }(), "3\n", "", true},
		{"confirm noninteractive", mismatch("confirm"), "", "", true},
		{"force interactive typed", func() Input {
			x := mismatch("block")
			x.Interactive = true
			x.Force = true
			x.ExplicitProfile = true
			return x
		}(), "personal\n", "personal", false},
		{"force interactive wrong", func() Input {
			x := mismatch("block")
			x.Interactive = true
			x.Force = true
			x.ExplicitProfile = true
			return x
		}(), "yes\n", "", true},
		{"force noninteractive explicit", func() Input { x := mismatch("block"); x.Force = true; x.ExplicitProfile = true; return x }(), "", "personal", false},
		{"force noninteractive implicit", func() Input { x := mismatch("block"); x.Force = true; return x }(), "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			profile, err := Check(tt.in, strings.NewReader(tt.input), &out)
			if errors.Is(err, ErrRefused) != tt.wantErr {
				t.Fatalf("error=%v output=%s", err, out.String())
			}
			if profile != tt.profile {
				t.Fatalf("profile=%q, want %q; output=%s", profile, tt.profile, out.String())
			}
			if tt.in.Expected != tt.in.Selected && !strings.Contains(out.String(), "Expected profile") {
				t.Fatal("missing mismatch explanation")
			}
		})
	}
}

func TestBidirectionalExposureText(t *testing.T) {
	in := mismatch("block")
	var out bytes.Buffer
	_, _ = Check(in, strings.NewReader(""), &out)
	if !strings.Contains(strings.ToLower(out.String()), "company source code") {
		t.Fatal(out.String())
	}
	in.Expected, in.Selected = "personal", "work"
	out.Reset()
	_, _ = Check(in, strings.NewReader(""), &out)
	if !strings.Contains(strings.ToLower(out.String()), "personal source code") {
		t.Fatal(out.String())
	}
}
