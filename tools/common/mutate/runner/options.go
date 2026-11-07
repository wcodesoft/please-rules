// Package runner orchestrates multi-language Tree-sitter mutation discovery,
// baseline verification, per-mutant test execution, and report generation.
package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tools/common/treesitter"
)

// Options configures a mutation testing run.
type Options struct {
	Srcs              []string
	Lang              string
	Target            string
	TestCmd           string
	WorkDir           string
	StripPrefix       string
	OutReport         string
	ResultsFile       string
	MinScore          float64
	TimeoutMultiplier float64
	MutantTimeout     time.Duration
	Mutators          []string
	ListOnly          bool
	Stdout            io.Writer
	Stderr            io.Writer
}

func normalizeOptions(opts Options) (Options, string, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.TimeoutMultiplier <= 0 {
		opts.TimeoutMultiplier = 5.0
	}
	workDir, err := resolveWorkDir(opts.WorkDir)
	if err != nil {
		return opts, "", err
	}
	return opts, workDir, nil
}

func resolveWorkDir(workDir string) (string, error) {
	if workDir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getwd: %w", err)
		}
		return dir, nil
	}
	dir, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("resolve work-dir: %w", err)
	}
	return dir, nil
}

func computeMutantTimeout(baseDur time.Duration, opts Options) time.Duration {
	if opts.MutantTimeout > 0 {
		return opts.MutantTimeout
	}
	scaled := time.Duration(float64(baseDur) * opts.TimeoutMultiplier)
	if scaled < 5*time.Second {
		return 5 * time.Second
	}
	return scaled
}

func parseMutatorFilter(specs []string) (map[treesitter.MutatorCategory]bool, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	valid := validMutatorSet()
	allowed := make(map[treesitter.MutatorCategory]bool)
	for _, part := range splitCommaSpecs(specs) {
		cat := treesitter.MutatorCategory(part)
		if !valid[cat] {
			return nil, fmt.Errorf("unknown mutator category %q", part)
		}
		allowed[cat] = true
	}
	return allowed, nil
}

func validMutatorSet() map[treesitter.MutatorCategory]bool {
	valid := make(map[treesitter.MutatorCategory]bool, len(treesitter.AllMutatorCategories))
	for _, c := range treesitter.AllMutatorCategories {
		valid[c] = true
	}
	return valid
}

func splitCommaSpecs(specs []string) []string {
	var out []string
	for _, spec := range specs {
		for _, part := range strings.Split(spec, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}
