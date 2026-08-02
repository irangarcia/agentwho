package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/paths"
)

func setupCurrentTest(t *testing.T) paths.Paths {
	t.Helper()
	root := t.TempDir()
	configHome := filepath.Join(root, "config")
	dataHome := filepath.Join(root, "data")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)
	p := paths.FromHomes(configHome, dataHome)
	if err := p.EnsureBase(); err != nil {
		t.Fatal(err)
	}
	c := config.Default()
	c.Profiles["work"] = config.Profile{Kind: "work"}
	if err := config.Save(p.Config, c); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCurrentPrintsOnlyProfileName(t *testing.T) {
	setupCurrentTest(t)
	t.Setenv("AGENTWHO_PROFILE", "")
	var output bytes.Buffer
	a := &app{in: strings.NewReader(""), out: &output, errout: &output}
	if err := a.currentCmd().Execute(); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "personal\n" {
		t.Fatalf("got %q, want stable plain output", got)
	}
}

func TestCurrentJSONIsVersionedAndReportsOverride(t *testing.T) {
	setupCurrentTest(t)
	t.Setenv("AGENTWHO_PROFILE", "work")
	var output bytes.Buffer
	a := &app{in: strings.NewReader(""), out: &output, errout: &output}
	cmd := a.currentCmd()
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got currentOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Version != 1 || got.Profile != "work" || got.ExpectedProfile != "personal" || got.Source != "explicit" || !got.Mismatch {
		t.Fatalf("unexpected current JSON: %+v", got)
	}
}

func TestCurrentRejectsUnknownOverride(t *testing.T) {
	setupCurrentTest(t)
	t.Setenv("AGENTWHO_PROFILE", "missing")
	a := &app{in: strings.NewReader(""), out: &bytes.Buffer{}, errout: &bytes.Buffer{}}
	err := a.currentCmd().Execute()
	if err == nil || !strings.Contains(err.Error(), `current profile "missing" does not exist`) {
		t.Fatalf("got error %v", err)
	}
}
