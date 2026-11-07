package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCLIFlagsAndList(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "lib.rs")
	if err := os.WriteFile(srcPath, []byte("pub fn inc(x: i32) -> i32 { x + 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := runCLI([]string{"-srcs", srcPath, "-work-dir", dir, "-list"}, &stdout, &stderr); err != nil {
		t.Fatalf("runCLI -list: %v", err)
	}
	if !strings.Contains(stdout.String(), "(arithmetic) + -> -") {
		t.Fatalf("unexpected -list output:\n%s", stdout.String())
	}
}
