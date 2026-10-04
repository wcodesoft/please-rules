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
	"tools/please_ts/npmcache"
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
	npmDeps := buildVitestNpmDependencies(im)
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
	}
	if err := writeVitestPackageJSON(opts.VitestDir, coverageDir != "", npmDeps); err != nil {
		return fmt.Errorf("failed preparing the vitest dependencies: %w", err)
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
	// npm packages come from the merged cache only: a ts_npm_module missing from the target's
	// deps must fail, not be downloaded.
	if len(npmDeps) > 0 {
		denoArgs = append(denoArgs, "--cached-only")
	}
	if coverageDir != "" || len(npmDeps) > 0 {
		// The v8 coverage provider is a peer dependency that vitest resolves from the
		// project's node_modules, and so are the npm packages of the target (see
		// buildVitestNpmDependencies); deno only materializes them when asked to.
		denoArgs = append(denoArgs, "--node-modules-dir=auto")
	}
	vitestSpec := "npm:vitest"
	if opts.VitestDir != "" {
		// Pinned to the cached version: an unversioned specifier resolves to the registry's
		// latest release, which Deno would download into the shared (read-only) cache.
		version, err := npmcache.CachedVersion(opts.VitestDir, "vitest")
		if err != nil {
			_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
			return err
		}
		if version != "" {
			vitestSpec += "@" + version
		}
	}
	denoArgs = append(denoArgs, "-A", vitestSpec)
	denoArgs = append(denoArgs, args...)
	cmd := exec.Command(denoBin, denoArgs...)

	env := os.Environ()
	if opts.VitestDir != "" {
		denoDir, err := opts.vitestDenoDir(tmpDir, len(npmDeps) > 0)
		if err != nil {
			_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
			return err
		}
		env = append(env, "DENO_DIR="+denoDir)
	}
	if opts.BrowserBinary != "" {
		env = append(env, "PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH="+opts.BrowserBinary)
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	testErr := cmd.Run()
	if coverageDir != "" {
		if err := convertVitestCoverage(coverageDir, coverageFile, opts.LcovFile); err != nil && testErr == nil {
			testErr = err
		}
	}
	if _, err := os.Stat(resultsFile); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, testErr)
	}
	return testErr
}

// vitestDenoDir returns the DENO_DIR Vitest runs with. It is the shared Vitest cache itself,
// unless the target uses npm packages: then it is a per-run copy of that cache holding the cache
// slices of the target's ts_npm_module dependencies as well, so the shared cache is never written.
func (opts RunOptions) vitestDenoDir(tmpDir string, withNpm bool) (string, error) {
	shared, err := filepath.Abs(opts.VitestDir)
	if err != nil {
		shared = opts.VitestDir
	}
	if !withNpm {
		return shared, nil
	}
	denoDir := filepath.Join(tmpDir, ".deno_cache")
	if err := os.MkdirAll(denoDir, 0755); err != nil {
		return "", fmt.Errorf("failed creating DENO_DIR: %w", err)
	}
	if err := npmcache.Merge([]string{shared}, denoDir); err != nil {
		return "", fmt.Errorf("failed copying the Vitest cache: %w", err)
	}
	if _, err := npmcache.Prepare(".", denoDir); err != nil {
		return "", fmt.Errorf("failed preparing the npm cache: %w", err)
	}
	return denoDir, nil
}

// vitestCoverageArgs returns the vitest flags that collect lcov coverage into dir.
func vitestCoverageArgs(dir string) []string {
	return []string{"--coverage.enabled", "--coverage.provider=v8", "--coverage.reporter=lcov", "--coverage.reportsDirectory=" + dir}
}

// writeVitestPackageJSON declares in the working directory what deno must materialize in
// node_modules: the vitest coverage provider when coverage is on, and the npm packages of the
// target, at the versions that are cached. A package.json the target already ships is left as
// it is.
func writeVitestPackageJSON(vitestDir string, coverage bool, npmDeps map[string]string) error {
	deps := map[string]string{}
	if coverage {
		// the versions in the Vitest cache: "*" would mean the registry's latest release
		for _, name := range []string{"vitest", "@vitest/coverage-v8"} {
			version, err := npmcache.CachedVersion(vitestDir, name)
			if err != nil {
				return err
			}
			if version == "" {
				version = "*"
			}
			deps[name] = version
		}
	}
	for name, version := range npmDeps {
		deps[name] = version
	}
	if len(deps) == 0 {
		return nil
	}
	if _, err := os.Stat("package.json"); err == nil {
		return nil
	}
	data, err := json.Marshal(map[string]map[string]string{"dependencies": deps})
	if err != nil {
		return err
	}
	return os.WriteFile("package.json", append(data, '\n'), 0644)
}

// convertVitestCoverage turns the lcov report vitest wrote into dir into the
// Cobertura XML file that Please reads, and also exports it as raw lcov to lcovFile.
func convertVitestCoverage(dir, outputFile, lcovFile string) error {
	lcov, err := os.ReadFile(filepath.Join(dir, "lcov.info"))
	if err != nil {
		return fmt.Errorf("vitest produced no lcov coverage: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return err
	}
	cwd, _ := os.Getwd()
	if err := os.WriteFile(outputFile, lcovToCoberturaXML(lcov, cwd), 0644); err != nil {
		return err
	}
	return writeRawLcov(lcovFile, lcov, cwd)
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
		// npm packages are not files: Deno imports them (see buildVitestNpmImports).
		if strings.HasPrefix(target, "npm:") {
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

// buildVitestNpmDependencies lists the npm packages (ts_npm_module) of the import map as
// package.json dependencies, name to exact version. Vite cannot resolve an npm: specifier, and
// Vitest cannot import one (Deno refuses it in the loader Vitest runs under). With these in a
// package.json, `deno run --node-modules-dir=auto` materializes a node_modules directory in the
// test's working directory from the merged cache, offline, and Vite resolves and externalizes
// the packages as it does any others: exports, CommonJS and subpaths included.
func buildVitestNpmDependencies(im *importmap.ImportMap) map[string]string {
	deps := map[string]string{}
	if im == nil {
		return deps
	}
	for k, target := range im.Imports {
		spec, ok := strings.CutPrefix(target, "npm:")
		if !ok || strings.HasSuffix(k, "/") {
			continue
		}
		// npm:name@version or npm:@scope/name@version
		at := strings.LastIndex(spec, "@")
		if at <= 0 {
			continue
		}
		deps[spec[:at]] = spec[at+1:]
	}
	return deps
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
