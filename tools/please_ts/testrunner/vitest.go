package testrunner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tools/please_ts/importmap"
)

func (opts RunOptions) runVitest(resultsFile string) error {
	tmpDir, err := os.MkdirTemp("", "please_ts_vitest_*")
	if err != nil {
		return fmt.Errorf("failed creating temp dir for vitest: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Synthesize target-local import map for alias resolution
	im, err := importmap.Synthesize(opts.ModuleName, opts.Srcs, opts.Deps, ".")
	if err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed synthesizing import map: %w", err)
	}

	aliasMap := buildVitestAliases(im)
	resolvedSrcs := resolveSources(opts.Srcs)

	configPath := filepath.Join(tmpDir, "vitest.config.mjs")
	if err := writeVitestConfig(configPath, resolvedSrcs, aliasMap); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed writing vitest config: %w", err)
	}

	absResults, err := filepath.Abs(resultsFile)
	if err != nil {
		absResults = resultsFile
	}

	args := []string{
		"run",
		"--config", configPath,
		"--reporter=junit",
		"--outputFile=" + absResults,
	}

	if opts.Browser != "" {
		args = append(args, "--browser.name="+opts.Browser, "--browser.headless")
	}

	coverageActive, coverageFile := opts.resolveCoverage()
	coverageDir := ""
	if coverageActive && coverageFile != "" {
		// Vitest empties its reports directory before each run, so it must not be the
		// test working directory (the default location of the coverage file).
		coverageDir = filepath.Join(tmpDir, "coverage")
		args = append(args, vitestCoverageArgs(coverageDir)...)
		if err := writeCoveragePackageJSON(); err != nil {
			return fmt.Errorf("failed preparing vitest coverage provider: %w", err)
		}
	}

	args = append(args, opts.ExtraArgs...)
	args = append(args, resolvedSrcs...)

	denoBin := opts.Deno
	if denoBin == "" {
		denoBin = "deno"
	}
	denoArgs := []string{"run"}
	if opts.VitestDir != "" {
		denoArgs = append(denoArgs, "--no-remote")
	}
	if coverageDir != "" {
		// The v8 coverage provider is a peer dependency that vitest resolves from the
		// project's node_modules; deno only materializes it when asked to.
		denoArgs = append(denoArgs, "--node-modules-dir=auto")
	}
	denoArgs = append(denoArgs, "-A", "npm:vitest")
	denoArgs = append(denoArgs, args...)
	cmd := exec.Command(denoBin, denoArgs...)

	env := os.Environ()
	if opts.VitestDir != "" {
		if absV, err := filepath.Abs(opts.VitestDir); err == nil {
			env = append(env, "DENO_DIR="+absV)
		} else {
			env = append(env, "DENO_DIR="+opts.VitestDir)
		}
	}
	if opts.BrowserBinary != "" {
		env = append(env, "PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH="+opts.BrowserBinary)
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	testErr := cmd.Run()
	if coverageDir != "" {
		if err := convertVitestCoverage(coverageDir, coverageFile); err != nil && testErr == nil {
			testErr = err
		}
	}
	if _, err := os.Stat(resultsFile); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, testErr)
	}
	return testErr
}

// vitestCoverageArgs returns the vitest flags that collect lcov coverage into dir.
func vitestCoverageArgs(dir string) []string {
	return []string{"--coverage.enabled", "--coverage.provider=v8", "--coverage.reporter=lcov", "--coverage.reportsDirectory=" + dir}
}

// writeCoveragePackageJSON declares the vitest coverage provider in the working
// directory unless the target already ships a package.json.
func writeCoveragePackageJSON() error {
	if _, err := os.Stat("package.json"); err == nil {
		return nil
	}
	return os.WriteFile("package.json", []byte(`{"dependencies":{"vitest":"*","@vitest/coverage-v8":"*"}}`+"\n"), 0644)
}

// convertVitestCoverage turns the lcov report vitest wrote into dir into the
// Cobertura XML file that Please reads.
func convertVitestCoverage(dir, outputFile string) error {
	lcov, err := os.ReadFile(filepath.Join(dir, "lcov.info"))
	if err != nil {
		return fmt.Errorf("vitest produced no lcov coverage: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return err
	}
	cwd, _ := os.Getwd()
	return os.WriteFile(outputFile, lcovToCoberturaXML(lcov, cwd), 0644)
}

func buildVitestAliases(im *importmap.ImportMap) map[string]string {
	aliasMap := make(map[string]string)
	if im == nil {
		return aliasMap
	}
	for k, target := range im.Imports {
		if strings.HasSuffix(k, "/") {
			continue
		}
		if k == "vitest" || strings.HasPrefix(k, "vitest/") || k == "chai" || strings.HasPrefix(k, "chai/") {
			continue
		}
		tClean := strings.TrimSuffix(target, "/")
		if tClean != "" {
			if absTarget, err := filepath.Abs(tClean); err == nil {
				aliasMap[k] = absTarget
			} else {
				aliasMap[k] = tClean
			}
		}
	}
	return aliasMap
}

// aliasEntry is one Vite resolve.alias entry in the generated config.
type aliasEntry struct {
	Find        string `json:"find"`
	Replacement string `json:"replacement"`
}

// writeVitestConfig writes a Vitest config whose aliases are an ordered array of
// exact-match entries. A plain object alias prefix-matches ("a/b" is rewritten by
// the alias "a"), which breaks nested module names such as "@scope/app" and
// "@scope/app/components/widget"; subpaths are already enumerated by the import map.
func writeVitestConfig(configPath string, srcs []string, aliases map[string]string) error {
	keys := make([]string, 0, len(aliases))
	for k := range aliases {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	entries := make([]aliasEntry, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, aliasEntry{Find: "^" + regexp.QuoteMeta(k) + "$", Replacement: aliases[k]})
	}
	aliasBytes, _ := json.MarshalIndent(entries, "    ", "  ")
	srcsBytes, _ := json.Marshal(srcs)

	// "find" is serialized as a string and revived into a RegExp by the config.
	configContent := fmt.Sprintf(`const aliases = %s;

export default {
  test: {
    globals: true,
    include: %s,
    watch: false,
  },
  resolve: {
    alias: aliases.map((a) => ({ find: new RegExp(a.find), replacement: a.replacement })),
  },
};
`, string(aliasBytes), string(srcsBytes))

	return os.WriteFile(configPath, []byte(configContent), 0644)
}
