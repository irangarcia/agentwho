package resolve

import (
	"testing"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/gitctx"
)

func TestRulePrecedence(t *testing.T) {
	c := config.Config{Version: 1, Defaults: config.Defaults{Profile: "personal", Enforcement: "confirm"}, Profiles: map[string]config.Profile{"personal": {Kind: "personal"}, "work": {Kind: "work"}}, Rules: []config.Rule{
		{Match: config.Match{Path: "/code"}, Profile: "personal", Enforcement: "confirm"},
		{Match: config.Match{GitOrganization: "github.com/acme"}, Profile: "work", Enforcement: "block"},
		{Match: config.Match{GitRemote: "github.com/acme/backend"}, Profile: "personal", Enforcement: "block"},
		{Match: config.Match{Path: "/code/acme"}, Profile: "work", Enforcement: "confirm"},
	}}
	ctx := gitctx.Context{Directory: "/code/acme/backend", Remote: "github.com/acme/backend", Organization: "github.com/acme"}
	if got := Resolve(c, ctx); got.Expected != "personal" || got.RuleIndex != 2 {
		t.Fatalf("exact remote did not win: %+v", got)
	}
	ctx.Remote = "github.com/acme/other"
	if got := Resolve(c, ctx); got.Expected != "work" || got.RuleIndex != 1 {
		t.Fatalf("organization did not win: %+v", got)
	}
	ctx.Organization = "github.com/other"
	if got := Resolve(c, ctx); got.Expected != "work" || got.RuleIndex != 3 {
		t.Fatalf("longest path did not win: %+v", got)
	}
	ctx.Directory = "/elsewhere"
	if got := Resolve(c, ctx); got.Expected != "personal" || got.RuleIndex != -1 {
		t.Fatalf("default did not win: %+v", got)
	}
}

func TestEquivalent(t *testing.T) {
	a := config.Rule{Match: config.Match{GitRemote: "host/a/b"}, Profile: "work"}
	b := config.Rule{Match: config.Match{GitRemote: "host/a/b"}, Profile: "personal"}
	if !Equivalent(a, b) {
		t.Fatal("matcher equivalence should ignore target")
	}
}
