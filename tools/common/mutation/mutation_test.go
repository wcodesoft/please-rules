package mutation

import "testing"

func TestRecomputeAndScore(t *testing.T) {
	r := &Report{
		Language: "go",
		Target:   "//pkg:lib_mutate_test",
		MinScore: 70.0,
		Mutants: []Mutant{
			{
				File:        "pkg/lib.go",
				Line:        15,
				Col:         8,
				Mutator:     "comparison",
				Function:    "Clamp",
				Original:    ">",
				Replacement: ">=",
				Status:      Survived,
				DurationMs:  95,
			},
			{
				File:        "pkg/lib.go",
				Line:        12,
				Col:         10,
				Mutator:     "arithmetic",
				Function:    "Add",
				Original:    "+",
				Replacement: "-",
				Status:      Killed,
				DurationMs:  142,
			},
			{
				File:        "pkg/lib.go",
				Line:        20,
				Col:         4,
				Mutator:     "logical",
				Function:    "Check",
				Original:    "&&",
				Replacement: "||",
				Status:      Timeout,
				DurationMs:  5000,
			},
			{
				File:        "pkg/lib.go",
				Line:        25,
				Col:         4,
				Mutator:     "arithmetic",
				Function:    "Broken",
				Original:    "+",
				Replacement: "-",
				Status:      Unviable,
				DurationMs:  10,
			},
		},
	}

	r.Recompute()

	if r.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", r.SchemaVersion)
	}
	if r.Total != 3 {
		t.Fatalf("Total = %d, want 3 (unviable excluded from evaluated total)", r.Total)
	}
	if r.Killed != 1 || r.Survived != 1 || r.Timeout != 1 || r.Unviable != 1 {
		t.Fatalf("counts = (k=%d, s=%d, t=%d, u=%d), want (1, 1, 1, 1)",
			r.Killed, r.Survived, r.Timeout, r.Unviable)
	}
	if r.Score != 66.67 {
		t.Fatalf("Score = %.2f, want 66.67", r.Score)
	}
	if r.Mutants[0].Line != 12 || r.Mutants[0].ID != "m1" {
		t.Fatalf("first mutant = %+v, want line 12 id m1", r.Mutants[0])
	}

	if err := r.CheckThreshold(70.0); err == nil {
		t.Fatal("expected threshold error for 66.67% < 70.0%, got nil")
	}
	if err := r.CheckThreshold(60.0); err != nil {
		t.Fatalf("unexpected threshold error for 66.67%% >= 60.0%%: %v", err)
	}
	if err := r.CheckSurvived(); err == nil {
		t.Fatal("expected CheckSurvived error when Survived > 0, got nil")
	}
}

func TestMergeAndValidStatus(t *testing.T) {
	if !ValidStatus(Killed) || !ValidStatus(Survived) || ValidStatus("invalid") {
		t.Fatal("unexpected ValidStatus result")
	}

	r1 := &Report{
		Language: "ts",
		Mutants: []Mutant{
			{File: "pkg/a.ts", Line: 5, Col: 2, Mutator: "equality", Original: "===", Replacement: "!==", Status: Survived},
		},
	}
	r2 := &Report{
		Language: "ts",
		Mutants: []Mutant{
			{File: "pkg/a.ts", Line: 5, Col: 2, Mutator: "equality", Original: "===", Replacement: "!==", Status: Killed},
		},
	}
	merged := Merge(r1, r2)
	if len(merged.Mutants) != 1 || merged.Mutants[0].Status != Killed {
		t.Fatalf("merged = %+v, want 1 killed mutant", merged)
	}
}
