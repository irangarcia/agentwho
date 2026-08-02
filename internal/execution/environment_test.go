package execution

import (
	"os"
	"testing"
)

func TestExecutionEnvironmentReflectsDecisionAndDropsForce(t *testing.T) {
	got := executionEnvironment([]string{
		"PATH=/bin",
		"AGENTWHO_PROFILE=personal",
		"AGENTWHO_FORCE=1",
		"KEEP=yes",
	}, "work")
	want := map[string]bool{
		"PATH=/bin":             true,
		"KEEP=yes":              true,
		"AGENTWHO_PROFILE=work": true,
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected environment: %v", got)
	}
	for _, item := range got {
		if !want[item] {
			t.Errorf("unexpected item %q in %v", item, got)
		}
	}
}

func TestDevNullIsNotInteractiveInput(t *testing.T) {
	file, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if interactiveInput(file) {
		t.Fatal("/dev/null was incorrectly treated as interactive input")
	}
}
