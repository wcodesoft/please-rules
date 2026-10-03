package testrunner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"tools/common/lcov"
)

// buildRawLcov turns llvm-cov's lcov export into the raw report kept for consumers:
// repository-relative paths, the synthetic test runner dropped, and, from the JSON
// export (llvmJSON, may be empty), synthesized branch records for lines that ran
// only partly. Swift emits no branch regions, so the lcov export itself has function
// records but no branches.
func buildRawLcov(lcovData, llvmJSON []byte, cwd string) ([]byte, error) {
	report, err := lcov.Parse(bytes.NewReader(lcovData))
	if err != nil {
		return nil, fmt.Errorf("parsing lcov report: %w", err)
	}
	cleanCwd := filepath.Clean(cwd)
	normalize := func(p string) string { return coveragePath(p, cleanCwd) }

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

// exportRawLcov writes the raw lcov report. The JSON export adds branch records for
// partly executed lines; if it fails the report is still written without them.
func exportRawLcov(opts RunOptions, testBinary, mergedProfdata string, lcovData []byte, cwd string) error {
	var jsonBuf, errBuf bytes.Buffer
	cmd := exec.Command(opts.LlvmCov, "export", "-format=text", testBinary, "-instr-profile="+mergedProfdata)
	cmd.Stdout = &jsonBuf
	cmd.Stderr = &errBuf
	var llvmJSON []byte
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: no sub-line regions for the raw lcov export: %v: %s\n", err, errBuf.String())
	} else {
		llvmJSON = jsonBuf.Bytes()
	}
	data, err := buildRawLcov(lcovData, llvmJSON, cwd)
	if err != nil {
		return err
	}
	return writeRawLcov(opts.LcovFile, data)
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
