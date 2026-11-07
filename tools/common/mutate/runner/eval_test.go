package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tools/common/mutation"
)

func TestEvaluateMutantAndRestore(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "val.txt")
	orig := []byte("original")
	if err := os.WriteFile(srcPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	fi := fileInput{
		absPath: srcPath,
		relPath: "val.txt",
		orig:    orig,
		mode:    0o644,
	}

	status, _, err := evaluateMutant(context.Background(), fi, []byte("mutated"), "grep -q original val.txt", dir, 5*time.Second)
	if err != nil {
		t.Fatalf("evaluateMutant: %v", err)
	}
	if status != mutation.Killed {
		t.Fatalf("status = %s, want killed", status)
	}

	restored, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "original" {
		t.Fatalf("restored = %q, want original", string(restored))
	}
}
