package main

import (
	"testing"
)

func TestMainCommandExecution(t *testing.T) {
	if err := runCommand([]string{"--help"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
