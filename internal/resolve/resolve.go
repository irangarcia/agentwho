package resolve

import (
	"sort"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/gitctx"
)

type Result struct {
	Expected       string       `json:"expected_profile"`
	Enforcement    string       `json:"enforcement"`
	Matched        *config.Rule `json:"matched_rule,omitempty"`
	RuleIndex      int          `json:"-"`
	Specificity    string       `json:"specificity"`
	SpecificityNum int          `json:"-"`
}

func Resolve(c config.Config, ctx gitctx.Context) Result {
	best := Result{Expected: c.Defaults.Profile, Enforcement: c.Defaults.Enforcement, RuleIndex: -1, Specificity: "default"}
	for i := range c.Rules {
		r := c.Rules[i]
		score, label := match(r, ctx)
		if score > best.SpecificityNum {
			copy := r
			best = Result{Expected: r.Profile, Enforcement: r.Enforcement, Matched: &copy, RuleIndex: i, Specificity: label, SpecificityNum: score}
		}
	}
	return best
}

func match(r config.Rule, ctx gitctx.Context) (int, string) {
	switch {
	case r.Match.GitRemote != "" && r.Match.GitRemote == ctx.Remote:
		return 300000, "exact repository"
	case r.Match.GitOrganization != "" && r.Match.GitOrganization == ctx.Organization:
		return 200000, "git organization"
	case r.Match.Path != "" && gitctx.PathContains(r.Match.Path, ctx.Directory):
		return 100000 + len(r.Match.Path), "directory tree"
	default:
		return 0, ""
	}
}

type ListedRule struct {
	Index       int         `json:"index"`
	Rule        config.Rule `json:"rule"`
	Specificity string      `json:"specificity"`
	Rank        int         `json:"-"`
}

func Ordered(rules []config.Rule) []ListedRule {
	out := make([]ListedRule, 0, len(rules))
	for i, r := range rules {
		rank, label := 0, ""
		switch {
		case r.Match.GitRemote != "":
			rank, label = 300000, "exact repository"
		case r.Match.GitOrganization != "":
			rank, label = 200000, "git organization"
		case r.Match.Path != "":
			rank, label = 100000+len(r.Match.Path), "directory tree"
		}
		out = append(out, ListedRule{Index: i, Rule: r, Specificity: label, Rank: rank})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Rank > out[j].Rank })
	return out
}

func Equivalent(a, b config.Rule) bool {
	at, av := a.Match.TypeValue()
	bt, bv := b.Match.TypeValue()
	return at == bt && av == bv
}
