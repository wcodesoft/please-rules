package rules

import (
	"testing"

	golang "github.com/odvcencio/gotreesitter/grammars/go"
)

func TestCategoriesAndParse(t *testing.T) {
	if len(AllMutatorCategories) != 7 {
		t.Fatalf("len(AllMutatorCategories) = %d, want 7", len(AllMutatorCategories))
	}

	var empty LanguageConfig
	if _, _, err := empty.Parse([]byte("package p")); err == nil {
		t.Fatal("expected error when Grammar is nil")
	}

	cfg := &LanguageConfig{
		ID:      LangGo,
		Grammar: golang.Language,
	}
	tree, lang, err := cfg.Parse([]byte("package p\nfunc f() {}\n"))
	if err != nil || tree == nil || lang == nil {
		t.Fatalf("Parse failed: tree=%v, lang=%v, err=%v", tree, lang, err)
	}
}
