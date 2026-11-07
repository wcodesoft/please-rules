package rules

import (
	"strings"
	"testing"

	golang "github.com/odvcencio/gotreesitter/grammars/go"
)

func TestDiscoverAndApply(t *testing.T) {
	cfg := &LanguageConfig{
		ID:      LangGo,
		Grammar: golang.Language,
		Function: FunctionRules{
			NodeTypes: []string{"function_declaration"},
			NameField: "name",
			BodyField: "body",
			BodyTypes: []string{"block"},
		},
		Replacement: ReplacementRules{
			BinaryNodes: []string{"binary_expression"},
			BinaryOps: map[string]OpReplacement{
				"&&": {MutatorLogical, []string{"||"}},
				"+":  {MutatorArithmetic, []string{"-"}},
			},
		},
	}

	src := []byte("package demo\nfunc f(x int) bool { return x > 0 && x + 1 == 2 }\n")
	onlyLogical := map[MutatorCategory]bool{MutatorLogical: true}

	sites, err := cfg.Discover("demo.go", src, onlyLogical)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(sites) != 1 || sites[0].Mutator != MutatorLogical {
		t.Fatalf("expected 1 logical site, got: %+v", sites)
	}

	mutated := string(Apply(src, sites[0]))
	if !strings.Contains(mutated, "x > 0 || x + 1 == 2") {
		t.Fatalf("unexpected Apply output: %q", mutated)
	}
}
