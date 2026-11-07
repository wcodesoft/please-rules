package golang

import (
	"strings"
	"testing"

	"tools/common/treesitter/rules"
)

func TestConfig(t *testing.T) {
	src := []byte(`package calc

import "fmt"

const TopLevelConst = 1 + 2

func Clamp(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi && true {
		return hi
	}
	x++
	x += 1
	return x + 0
}

func TestClamp(t *testing.T) {
	if Clamp(5, 0, 10) != 5 {
		fmt.Println("fail")
	}
}
`)

	cfg := Config()
	if !cfg.IsTestFile("clamp_test.go") || cfg.IsTestFile("clamp.go") {
		t.Fatal("unexpected Go IsTestFile result")
	}

	sites, err := cfg.Discover("calc.go", src, nil)
	if err != nil {
		t.Fatalf("Config().Discover: %v", err)
	}
	if len(sites) == 0 {
		t.Fatal("expected mutation sites in Clamp, got 0")
	}

	for _, s := range sites {
		if s.Function != "Clamp" {
			t.Errorf("site function = %q, want Clamp", s.Function)
		}
		if s.Line == 5 {
			t.Errorf("unexpected mutation on package-level const line 5: %+v", s)
		}
	}

	foundLt := false
	for _, s := range sites {
		if s.Original == "<" && s.Replacement == "<=" {
			foundLt = true
			if !strings.Contains(string(rules.Apply(src, s)), "if x <= lo") {
				t.Fatalf("Apply did not produce expected 'if x <= lo'")
			}
		}
	}
	if !foundLt {
		t.Fatalf("did not find '<' -> '<=' mutation in %+v", sites)
	}
}
