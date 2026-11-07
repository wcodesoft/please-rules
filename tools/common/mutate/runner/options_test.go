package runner

import (
	"testing"
	"time"

	"tools/common/treesitter"
)

func TestNormalizeOptionsAndMutatorFilter(t *testing.T) {
	opts, workDir, err := normalizeOptions(Options{})
	if err != nil {
		t.Fatalf("normalizeOptions: %v", err)
	}
	if opts.TimeoutMultiplier != 5.0 || opts.Stdout == nil || opts.Stderr == nil || workDir == "" {
		t.Fatalf("unexpected normalized options: %+v, workDir=%q", opts, workDir)
	}

	if got := computeMutantTimeout(100*time.Millisecond, opts); got != 5*time.Second {
		t.Fatalf("computeMutantTimeout floor = %v, want 5s", got)
	}
	if got := computeMutantTimeout(2*time.Second, opts); got != 10*time.Second {
		t.Fatalf("computeMutantTimeout scaled = %v, want 10s", got)
	}

	allowed, err := parseMutatorFilter([]string{"arithmetic,logical"})
	if err != nil {
		t.Fatalf("parseMutatorFilter: %v", err)
	}
	if !allowed[treesitter.MutatorArithmetic] || !allowed[treesitter.MutatorLogical] || len(allowed) != 2 {
		t.Fatalf("unexpected allowed mutators: %+v", allowed)
	}

	if _, err := parseMutatorFilter([]string{"nonexistent"}); err == nil {
		t.Fatal("expected error for unknown mutator category")
	}
}
