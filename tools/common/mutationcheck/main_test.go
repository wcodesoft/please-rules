package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/common/mutation"
)

func TestMutationCheck(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "test.mutation.json")

	rep := &mutation.Report{
		Language: "go",
		Target:   "//pkg:lib_mutate_test",
		MinScore: 50,
		Mutants: []mutation.Mutant{
			{
				File:        "pkg/lib.go",
				Line:        10,
				Col:         5,
				Mutator:     "arithmetic",
				Function:    "Add",
				Original:    "+",
				Replacement: "-",
				Status:      mutation.Killed,
			},
			{
				File:        "pkg/lib.go",
				Line:        20,
				Col:         8,
				Mutator:     "comparison",
				Function:    "Clamp",
				Original:    ">",
				Replacement: ">=",
				Status:      mutation.Survived,
			},
		},
	}

	f, err := os.Create(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := rep.WriteJSON(f); err != nil {
		t.Fatal(err)
	}
	f.Close()

	var stdout bytes.Buffer
	err = run([]string{
		"-report", reportPath,
		"-lang", "go",
		"-min-score", "50",
		"-min-total", "2",
		"-min-killed", "1",
		"-path", "pkg/lib.go",
		"-line", "10=killed",
		"-line", "20=survived",
		"-function", "Add=killed",
		"-function", "Clamp=survived",
	}, &stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "all assertions hold") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}

	// Failing assertions
	err = run([]string{
		"-report", reportPath,
		"-min-score", "80",
		"-path", "pkg/lib.go",
		"-line", "10=survived",
	}, &stdout)
	if err == nil || !strings.Contains(err.Error(), "2 assertion(s) failed") {
		t.Fatalf("expected 2 assertion failures, got: %v", err)
	}
}
