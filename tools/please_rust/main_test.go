package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	if !strings.Contains(buf.String(), "Usage: plz_rust") {
		t.Errorf("expected usage output to contain 'Usage: plz_rust', got %s", buf.String())
	}
}

func TestRunCLI_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCLI(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when no args provided, got nil")
	}
	if !strings.Contains(stderr.String(), "Usage: plz_rust") {
		t.Errorf("expected stderr to contain usage, got %s", stderr.String())
	}
}

func TestRunCLI_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCLI([]string{"unknown_cmd"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
	if !strings.Contains(stderr.String(), "Unknown command: unknown_cmd") {
		t.Errorf("expected stderr to report unknown command, got %s", stderr.String())
	}
}
