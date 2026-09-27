package testrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tools/please_ts/importmap"
)

func (opts RunOptions) runDeno(resultsFile string) error {
	denoBin := opts.Deno
	if denoBin == "" {
		denoBin = "deno"
	}

	coverageActive, coverageFile := opts.resolveCoverage()

	tmpDir, err := os.MkdirTemp("", "please_ts_test_*")
	if err != nil {
		return fmt.Errorf("failed creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	denoCacheDir := filepath.Join(tmpDir, ".deno_cache")
	if err := os.MkdirAll(denoCacheDir, 0755); err != nil {
		return fmt.Errorf("failed creating DENO_DIR: %w", err)
	}

	// 1. Synthesize target-local import map
	importMapPath := ".import_map.json"
	im, err := importmap.Synthesize(opts.ModuleName, opts.Srcs, opts.Deps, ".")
	if err != nil {
		return fmt.Errorf("failed synthesizing import map: %w", err)
	}
	if err := im.WriteToFile(importMapPath); err != nil {
		return fmt.Errorf("failed writing import map: %w", err)
	}
	defer os.Remove(importMapPath)

	// 2. Prepare test arguments
	resolvedSrcs := resolveSources(opts.Srcs)

	args := []string{
		"test",
		"--no-remote",
		"--unstable-detect-cjs",
		"--allow-read",
		"--allow-env",
		"--import-map", importMapPath,
		"--junit-path", resultsFile,
	}

	covDir := ""
	if coverageActive {
		covDir = filepath.Join(tmpDir, "cov_profile")
		args = append(args, "--coverage="+covDir)
	}

	args = append(args, opts.ExtraArgs...)
	args = append(args, resolvedSrcs...)

	cmd := exec.Command(denoBin, args...)
	cmd.Env = append(os.Environ(), "DENO_DIR="+denoCacheDir)

	nodePath := buildNodePath(opts.Deps)
	if nodePath != "" {
		cmd.Env = append(cmd.Env, "NODE_PATH="+nodePath)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	testErr := cmd.Run()

	// 3. Ensure JUnit XML exists even if test failed early
	if _, err := os.Stat(resultsFile); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, testErr)
	}

	// 4. Generate coverage report if requested
	if coverageActive && coverageFile != "" && covDir != "" {
		_ = generateDenoCoverage(denoBin, covDir, coverageFile, denoCacheDir)
	}

	if testErr != nil {
		return fmt.Errorf("tests failed: %w", testErr)
	}
	return nil
}

func buildNodePath(deps []string) string {
	seen := make(map[string]bool)
	var nodePaths []string

	addPath := func(p string) {
		abs, err := filepath.Abs(p)
		if err != nil {
			return
		}
		if !seen[abs] {
			seen[abs] = true
			nodePaths = append(nodePaths, abs)
		}
	}

	for _, dep := range deps {
		if info, err := os.Stat(dep); err == nil && info.IsDir() {
			addPath(dep)
			addPath(filepath.Dir(dep))
		}
	}

	addPath(".")

	// Walk cwd to discover module roots (containing package.json or ts_metadata.json)
	// and add their parent directories, self, and any .deps directory to NODE_PATH.
	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			if info.Name() == "package.json" || info.Name() == "ts_metadata.json" {
				dir := filepath.Dir(path)
				parent := filepath.Dir(dir)
				addPath(parent)
				addPath(dir)
				depsDir := filepath.Join(dir, ".deps")
				if dInfo, dErr := os.Stat(depsDir); dErr == nil && dInfo.IsDir() {
					addPath(depsDir)
				}
			}
		}
		return nil
	})

	if len(nodePaths) == 0 {
		return ""
	}
	return strings.Join(nodePaths, ":")
}

func resolveSources(srcs []string) []string {
	var resolvedSrcs []string
	for _, s := range srcs {
		if _, err := os.Stat(s); err == nil {
			resolvedSrcs = append(resolvedSrcs, s)
			continue
		}
		found := false
		_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if !found && !info.IsDir() && (path == s || filepath.Base(path) == s || strings.HasSuffix(path, "/"+s)) {
				resolvedSrcs = append(resolvedSrcs, path)
				found = true
			}
			return nil
		})
		if !found {
			resolvedSrcs = append(resolvedSrcs, s)
		}
	}
	return resolvedSrcs
}
