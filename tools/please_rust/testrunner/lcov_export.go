package testrunner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"tools/common/lcov"
)

// buildRawLcov turns llvm-cov's lcov export into the raw report kept for consumers:
// repository-relative paths, external and toolchain files dropped, and, from the
// JSON export (llvmJSON, may be empty), synthesized branch records for lines that
// ran only partly. Stable Rust emits no branch regions, so the lcov export itself
// has function records but no branches.
func buildRawLcov(lcovData, llvmJSON []byte, repoRoot string) ([]byte, error) {
	report, err := lcov.Parse(bytes.NewReader(lcovData))
	if err != nil {
		return nil, fmt.Errorf("parsing lcov report: %w", err)
	}
	normalize := func(p string) string { return normalizePath(p, repoRoot) }

	kept := report.Files[:0]
	for _, f := range report.Files {
		if normalize(f.Path) != "" {
			kept = append(kept, f)
		}
	}
	report.Files = kept
	report.NormalizePaths(normalize)

	if len(llvmJSON) > 0 {
		if err := lcov.AddRegionBranches(report, bytes.NewReader(llvmJSON), normalize); err != nil {
			return nil, err
		}
	}

	var out bytes.Buffer
	if err := report.Write(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// writeRawLcov writes data to path. An empty path disables the export.
func writeRawLcov(path string, data []byte) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
