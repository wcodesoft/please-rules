package main

import (
	"bytes"
	"testing"
)

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	if !bytes.Contains(buf.Bytes(), []byte("please_wit")) {
		t.Errorf("expected usage to contain 'please_wit', got: %s", buf.String())
	}
}

func TestRunCommandUnknown(t *testing.T) {
	err := runCommand([]string{"unknown_subcmd"})
	if err == nil {
		t.Errorf("expected error for unknown subcommand, got nil")
	}
}

func TestRunCommandEmpty(t *testing.T) {
	err := runCommand([]string{})
	if err == nil {
		t.Errorf("expected error for empty args, got nil")
	}
}
