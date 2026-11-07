package runner

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/common/mutation"
)

func TestRunEndToEndAndRestore(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "calc.py")
	origContent := `def add(a, b):
    return a + b

def is_positive(x):
    return x > 0
`
	if err := os.WriteFile(srcPath, []byte(origContent), 0o644); err != nil {
		t.Fatal(err)
	}

	testScript := filepath.Join(dir, "check.sh")
	scriptContent := `#!/bin/sh
grep -q "return a + b" calc.py || exit 1
grep -q "return x >" calc.py || exit 1
exit 0
`
	if err := os.WriteFile(testScript, []byte(scriptContent), 0o755); err != nil {
		t.Fatal(err)
	}

	reportPath := filepath.Join(dir, "calc.mutation.json")
	junitPath := filepath.Join(dir, "test.results")
	var stdout, stderr bytes.Buffer

	rep, err := Run(context.Background(), Options{
		Srcs:        []string{"calc.py"},
		WorkDir:     dir,
		Target:      "//pkg:calc_mutate_test",
		TestCmd:     "./check.sh",
		OutReport:   reportPath,
		ResultsFile: junitPath,
		MinScore:    50,
		Stdout:      &stdout,
		Stderr:      &stderr,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\nstderr: %s", err, stderr.String())
	}

	afterBytes, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterBytes) != origContent {
		t.Fatalf("source file was not restored after mutation run:\ngot:\n%s\nwant:\n%s", string(afterBytes), origContent)
	}

	if rep.Total != 3 || rep.Killed != 2 || rep.Survived != 1 || rep.Score != 66.67 {
		t.Fatalf("unexpected report: %+v", rep)
	}

	f, err := os.Open(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	parsed, err := mutation.ParseJSON(f)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Score != 66.67 {
		t.Fatalf("parsed Score = %.2f, want 66.67", parsed.Score)
	}

	_, err = Run(context.Background(), Options{
		Srcs:     []string{"calc.py"},
		WorkDir:  dir,
		TestCmd:  "./check.sh",
		MinScore: 80,
		Stdout:   &stdout,
		Stderr:   &stderr,
	})
	if err == nil || !strings.Contains(err.Error(), "below minimum threshold") {
		t.Fatalf("expected min-score failure for 66.67 < 80, got: %v", err)
	}
}
