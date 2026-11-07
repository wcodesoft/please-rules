package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"tools/common/mutation"
	"tools/common/treesitter"
)

func runBaseline(ctx context.Context, opts Options, workDir string) (time.Duration, error) {
	if strings.TrimSpace(opts.TestCmd) == "" {
		return 0, fmt.Errorf("-test-cmd is required unless -list is set")
	}
	baseStart := time.Now()
	baseOut, baseStatus, err := runTestCommand(ctx, opts.TestCmd, workDir, 0)
	baseDur := time.Since(baseStart)
	if err != nil || baseStatus != mutation.Survived {
		return 0, fmt.Errorf("baseline test command failed before applying any mutations (%v):\n%s", err, baseOut)
	}
	return computeMutantTimeout(baseDur, opts), nil
}

func evaluateAllMutants(
	ctx context.Context,
	files []fileInput,
	testCmd string,
	workDir string,
	timeout time.Duration,
) ([]mutation.Mutant, error) {
	var all []mutation.Mutant
	for _, f := range files {
		fileMutants, err := evaluateFileMutants(ctx, f, len(all), testCmd, workDir, timeout)
		if err != nil {
			return nil, err
		}
		all = append(all, fileMutants...)
	}
	return all, nil
}

func evaluateFileMutants(
	ctx context.Context,
	f fileInput,
	startIdx int,
	testCmd string,
	workDir string,
	timeout time.Duration,
) ([]mutation.Mutant, error) {
	out := make([]mutation.Mutant, 0, len(f.sites))
	for i, s := range f.sites {
		id := fmt.Sprintf("m%d", startIdx+i+1)
		mutatedSrc := treesitter.Apply(f.orig, s)
		status, durMs, err := evaluateMutant(ctx, f, mutatedSrc, testCmd, workDir, timeout)
		if err != nil {
			return nil, err
		}
		out = append(out, siteToMutant(id, s, status, durMs))
	}
	return out, nil
}

func evaluateMutant(
	ctx context.Context,
	f fileInput,
	mutatedSrc []byte,
	testCmd string,
	workDir string,
	timeout time.Duration,
) (mutation.Status, int64, error) {
	if err := os.WriteFile(f.absPath, mutatedSrc, f.mode); err != nil {
		return "", 0, fmt.Errorf("write mutated %s: %w", f.relPath, err)
	}
	defer func() {
		_ = os.WriteFile(f.absPath, f.orig, f.mode)
	}()

	start := time.Now()
	_, status, _ := runTestCommand(ctx, testCmd, workDir, timeout)
	durMs := time.Since(start).Milliseconds()

	if err := os.WriteFile(f.absPath, f.orig, f.mode); err != nil {
		return "", durMs, fmt.Errorf("restore original %s: %w", f.relPath, err)
	}
	return status, durMs, nil
}

func runTestCommand(parentCtx context.Context, testCmd, workDir string, timeout time.Duration) (string, mutation.Status, error) {
	ctx := parentCtx
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(parentCtx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", testCmd)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), mutation.Timeout, ctx.Err()
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return string(out), mutation.Killed, err
		}
		return string(out), mutation.Unviable, err
	}
	return string(out), mutation.Survived, nil
}

func siteToMutant(id string, s treesitter.Site, status mutation.Status, durMs int64) mutation.Mutant {
	return mutation.Mutant{
		ID:          id,
		File:        s.File,
		Line:        s.Line,
		Col:         s.Col,
		EndLine:     s.EndLine,
		EndCol:      s.EndCol,
		Mutator:     string(s.Mutator),
		Function:    s.Function,
		Original:    s.Original,
		Replacement: s.Replacement,
		Status:      status,
		DurationMs:  durMs,
	}
}
