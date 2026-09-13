package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	out := buf.String()
	if !strings.Contains(out, "please_kotlin") {
		t.Errorf("usage missing please_kotlin: %s", out)
	}
	if !strings.Contains(out, "compile") {
		t.Errorf("usage missing compile: %s", out)
	}
	if !strings.Contains(out, "testrunner") {
		t.Errorf("usage missing testrunner: %s", out)
	}
}

func TestRunCommandUnknown(t *testing.T) {
	err := runCommand([]string{"nonexistent"})
	if err == nil {
		t.Errorf("expected error for unknown subcommand")
	}
}

func TestRunCommandEmpty(t *testing.T) {
	err := runCommand([]string{})
	if err == nil {
		t.Errorf("expected error for empty args")
	}
}

func TestRunCommandHelp(t *testing.T) {
	if err := runCommand([]string{"help"}); err != nil {
		t.Errorf("unexpected error on help: %v", err)
	}
}
