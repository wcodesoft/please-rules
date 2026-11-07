package mutation

import "testing"

func TestRelPathAndNormalizePaths(t *testing.T) {
	if got := RelPath("file:///work/pkg/a.ts", "/work"); got != "pkg/a.ts" {
		t.Fatalf("RelPath = %q, want pkg/a.ts", got)
	}
	if got := RelPath("/other/pkg/a.ts", "/work"); got != "/other/pkg/a.ts" {
		t.Fatalf("RelPath outside root = %q, want /other/pkg/a.ts", got)
	}

	r := &Report{
		Language: "ts",
		Mutants: []Mutant{
			{File: "/work/pkg/a.ts", Line: 5, Col: 2, Mutator: "equality", Original: "===", Replacement: "!==", Status: Killed},
		},
	}
	r.NormalizePaths(func(p string) string {
		return RelPath(p, "/work")
	})
	if r.Mutants[0].File != "pkg/a.ts" {
		t.Fatalf("Normalized file = %q, want pkg/a.ts", r.Mutants[0].File)
	}
}
