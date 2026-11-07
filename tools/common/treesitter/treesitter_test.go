package treesitter

import "testing"

func TestResolveLookupAndDiscoverAcrossSubpackages(t *testing.T) {
	cfg, err := ResolveLanguage("pkg/widget.tsx", "ts")
	if err != nil {
		t.Fatalf("ResolveLanguage(.tsx, ts): %v", err)
	}
	if cfg.ID != LangTSX || cfg.ReportName != "ts" {
		t.Fatalf("cfg = %+v, want LangTSX with ReportName ts", cfg)
	}

	if _, err := ResolveLanguage("pkg/unknown.xyz", ""); err == nil {
		t.Fatal("expected error for unknown extension, got nil")
	}
	if _, err := ResolveLanguage("pkg/file.go", "cobol"); err == nil {
		t.Fatal("expected error for unsupported explicit language, got nil")
	}

	if !IsTestFile("pkg/calc_test.go") || !IsTestFile("pkg/test_calc.py") || IsTestFile("pkg/calc.go") {
		t.Fatal("unexpected IsTestFile result")
	}

	sites, err := Discover(LangGo, "calc.go", []byte("package p\nfunc add(a, b int) int { return a + b }\n"), nil)
	if err != nil || len(sites) == 0 {
		t.Fatalf("Discover(LangGo): sites=%+v, err=%v", sites, err)
	}
}
