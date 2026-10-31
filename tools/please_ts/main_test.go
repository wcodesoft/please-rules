package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	output := buf.String()
	if !strings.Contains(output, "please_ts") {
		t.Errorf("expected please_ts in usage: %s", output)
	}
	if !strings.Contains(output, "compile") || !strings.Contains(output, "bundle") || !strings.Contains(output, "testrunner") || !strings.Contains(output, "freeze-cache") {
		t.Errorf("expected subcommands in usage: %s", output)
	}
}

func TestRunCommandHelp(t *testing.T) {
	if err := runCommand([]string{"--help"}); err != nil {
		t.Errorf("unexpected error for help: %v", err)
	}
}

func TestRunCommandUnknown(t *testing.T) {
	if err := runCommand([]string{"invalid_subcommand"}); err == nil {
		t.Errorf("expected error for unknown subcommand")
	}
}

func TestFreezeCacheNeedsTheDirectoryOfTheCache(t *testing.T) {
	if err := runCommand([]string{"freeze-cache"}); err == nil || !strings.Contains(err.Error(), "--deno-dir") {
		t.Errorf("expected an error that names the missing flag, got %v", err)
	}
}

func TestFreezeCacheCutsTheVersionsNewerThanTheCachedOnes(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "npm", "registry.npmjs.org", "ms")
	if err := os.MkdirAll(filepath.Join(pkg, "2.1.3"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `{"name":"ms","versions":{"2.1.3":{},"2.1.4":{}},"dist-tags":{"latest":"2.1.4"}}`
	if err := os.WriteFile(filepath.Join(pkg, "registry.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runCommand([]string{"freeze-cache", "--deno-dir", dir}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(pkg, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "2.1.4") {
		t.Errorf("expected 2.1.4 to be gone from the list, got %s", data)
	}
}
