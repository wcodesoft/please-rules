package binary

import (
	"testing"
)

func TestOptionsValidation(t *testing.T) {
	err := Run(Options{})
	if err == nil {
		t.Errorf("expected error when Main is empty")
	}

	err = Run(Options{Main: "main.ts"})
	if err == nil {
		t.Errorf("expected error when Out is empty")
	}
}
