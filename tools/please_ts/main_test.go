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
	if !strings.Contains(output, "please_ts") {
		t.Errorf("expected please_ts in usage: %s", output)
	}
	if !strings.Contains(output, "compile") || !strings.Contains(output, "bundle") || !strings.Contains(output, "testrunner") {
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
