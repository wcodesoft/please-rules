package typescript

import (
	"testing"

	"tools/common/treesitter/rules"
)

func TestConfigs(t *testing.T) {
	src := []byte(`export function isAdult(age: number): boolean {
  return age >= 18 && age === 21;
}

describe("isAdult", () => {
  it("works", () => {
    const x = 1 + 2;
  });
});
`)

	cfg := Config()
	if !cfg.IsTestFile("calc.test.ts") || !cfg.IsTestFile("calc.spec.tsx") || cfg.IsTestFile("calc.ts") {
		t.Fatal("unexpected TypeScript IsTestFile result")
	}

	sites, err := cfg.Discover("calc.ts", src, nil)
	if err != nil {
		t.Fatalf("Config().Discover: %v", err)
	}
	foundStrictEq := false
	for _, s := range sites {
		if s.Function == "isAdult" && s.Original == "===" && s.Replacement == "!==" {
			foundStrictEq = true
		}
	}
	if !foundStrictEq {
		t.Fatalf("expected '===' -> '!==' in isAdult, got: %+v", sites)
	}

	if TSXConfig().ID != rules.LangTSX || JavaScriptConfig().ID != rules.LangJavaScript {
		t.Fatal("unexpected TSXConfig or JavaScriptConfig ID")
	}
}
