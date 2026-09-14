package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	output := buf.String()

	if !strings.Contains(output, "please_swift") {
		t.Errorf("usage output should contain 'please_swift', got: %s", output)
	}
	if !strings.Contains(output, "compile") {
		t.Errorf("usage output should mention 'compile', got: %s", output)
	}
	if !strings.Contains(output, "binary") {
		t.Errorf("usage output should mention 'binary', got: %s", output)
	}
	if !strings.Contains(output, "testrunner") {
		t.Errorf("usage output should mention 'testrunner', got: %s", output)
	}
}

func TestRunCommandUnknown(t *testing.T) {
	err := runCommand([]string{"nonexistent-subcommand"})
	if err == nil {
		t.Error("expected error for unknown subcommand, got nil")
	}
}

func TestRunCommandNoArgs(t *testing.T) {
	err := runCommand([]string{})
	if err == nil {
		t.Error("expected error for empty args, got nil")
	}
}

func TestRunCommandHelp(t *testing.T) {
	err := runCommand([]string{"--help"})
	if err != nil {
		t.Errorf("expected no error for --help, got: %v", err)
	}
}
