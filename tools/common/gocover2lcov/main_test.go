package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const source = `package fx

func Used() int {
	return 1
}

func Unused() int {
	return 2
}
`

const profile = `mode: set
example.com/fx/pkg/lib.go:3.18,5.2 1 1
example.com/fx/pkg/lib.go:7.20,9.2 1 0
`

func setup(t *testing.T) (dir, profilePath string) {
	t.Helper()
	dir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "pkg"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pkg", "lib.go"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	profilePath = filepath.Join(dir, "cover.out")
	if err := os.WriteFile(profilePath, []byte(profile), 0644); err != nil {
		t.Fatal(err)
	}
	return dir, profilePath
}

func TestRunConvertsToStdout(t *testing.T) {
	dir, profilePath := setup(t)
	var out bytes.Buffer
	if err := run([]string{"-profile", profilePath, "-src-root", dir, "-strip-prefix", "example.com/fx"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"SF:pkg/lib.go\n", "FN:3,Used\n", "FNDA:1,Used\n", "FNDA:0,Unused\n", "DA:4,1\n", "DA:8,0\n"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}
}

func TestRunWritesOutputFile(t *testing.T) {
	dir, profilePath := setup(t)
	dest := filepath.Join(dir, "out.lcov")
	var out bytes.Buffer
	if err := run([]string{"-profile", profilePath, "-src-root", dir, "-strip-prefix", "example.com/fx", "-out", dest}, &out); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dest); !strings.Contains(string(got), "SF:pkg/lib.go\n") || out.Len() != 0 {
		t.Errorf("file = %q, stdout = %q", got, out.String())
	}
}

func TestRunErrors(t *testing.T) {
	var out bytes.Buffer
	if err := run(nil, &out); err == nil {
		t.Error("expected an error without -profile")
	}
	if err := run([]string{"-profile", "/nonexistent"}, &out); err == nil {
		t.Error("expected an error for a missing profile")
	}
}
