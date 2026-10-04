package compile

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tools/please_ts/importmap"
	"tools/please_ts/npmcache"
)

// Options holds configuration for TypeScript library compilation.
type Options struct {
	Deno       string
	Out        string
	Srcs       []string
	Deps       []string
	ModuleName string
	Flags      []string
	VitestDir  string
}

// Run executes the type-checking and packages the library output.
func Run(opts Options) error {
	if len(opts.Srcs) == 0 {
		return fmt.Errorf("no source files provided for compilation")
	}

	denoBin := opts.Deno
	if denoBin == "" {
		denoBin = "deno"
	}

	tmpDir, err := os.MkdirTemp("", "please_ts_compile_*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	denoCacheDir := filepath.Join(tmpDir, ".deno_cache")
	// npm packages provided by ts_npm_module targets are merged into the per-run Deno cache and
	// resolved from there, offline.
	usesNpmSlices := len(npmcache.Discover(".")) > 0
	switch {
	case opts.VitestDir != "" && !usesNpmSlices:
		// The shared Vitest cache is used as it is, and never written to by this target.
		if absV, err := filepath.Abs(opts.VitestDir); err == nil {
			denoCacheDir = absV
		} else {
			denoCacheDir = opts.VitestDir
		}
	default:
		if err := os.MkdirAll(denoCacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create DENO_DIR: %w", err)
		}
		if opts.VitestDir != "" {
			// A per-run copy of the Vitest cache, so the slices do not end up in the shared one.
			if err := npmcache.Merge([]string{opts.VitestDir}, denoCacheDir); err != nil {
				return fmt.Errorf("failed copying the Vitest cache: %w", err)
			}
		}
		if _, err := npmcache.Prepare(".", denoCacheDir); err != nil {
			return fmt.Errorf("failed preparing the npm cache: %w", err)
		}
	}

	// 1. Synthesize target-local import map
	importMapPath := ".import_map.json"
	im, err := importmap.Synthesize(opts.ModuleName, opts.Srcs, opts.Deps, ".")
	if err != nil {
		return fmt.Errorf("failed synthesizing import map: %w", err)
	}
	if opts.VitestDir != "" && im.Imports["vitest"] == "" {
		spec, err := vitestSpecifier(opts.VitestDir)
		if err != nil {
			return err
		}
		im.Imports["vitest"] = spec
	}
	if err := im.WriteToFile(importMapPath); err != nil {
		return fmt.Errorf("failed writing import map: %w", err)
	}
	defer os.Remove(importMapPath)

	// 2. Run `deno check` with --no-remote and target import map
	args := []string{
		"check",
		"--no-remote",
		"--import-map", importMapPath,
	}
	// npm packages come from the merged cache only: a ts_npm_module missing from the
	// target's deps must fail here, not be downloaded. (The shared Vitest cache alone is a
	// different mechanism and keeps its own behaviour.)
	if usesNpmSlices || (opts.VitestDir == "" && im.HasNpmSpecifiers()) {
		args = append(args, "--cached-only")
	}
	args = append(args, opts.Flags...)
	args = append(args, opts.Srcs...)

	cmd := exec.Command(denoBin, args...)
	cmd.Env = append(os.Environ(), "DENO_DIR="+denoCacheDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("deno check failed: %w", err)
	}

	// 3. Emit output
	if opts.Out != "" {
		if err := emitOutput(opts, im); err != nil {
			return fmt.Errorf("failed creating output: %w", err)
		}
	}

	return nil
}

// vitestSpecifier is the npm: specifier of the vitest in the Vitest cache at dir: pinned to the
// cached version, because an unversioned one resolves to the registry's latest release, which
// Deno would then download into that (read-only) cache.
func vitestSpecifier(dir string) (string, error) {
	version, err := npmcache.CachedVersion(dir, "vitest")
	if err != nil {
		return "", err
	}
	if version == "" {
		return "npm:vitest", nil
	}
	return "npm:vitest@" + version, nil
}

func emitOutput(opts Options, im *importmap.ImportMap) error {
	outDir := opts.Out
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	var files []string
	// Copy source files into outDir preserving structure
	for _, src := range opts.Srcs {
		destPath := filepath.Join(outDir, filepath.Base(src))
		if err := copyFile(src, destPath); err != nil {
			return fmt.Errorf("failed copying source %s to %s: %w", src, destPath, err)
		}
		files = append(files, filepath.Base(src))
	}

	// Determine entry point
	entryFile := ""
	if len(opts.Srcs) > 0 {
		cleanName := filepath.Base(opts.ModuleName)
		if strings.HasPrefix(cleanName, "@") {
			cleanName = strings.TrimPrefix(cleanName, "@")
		}
		for _, s := range opts.Srcs {
			b := filepath.Base(s)
			if b == cleanName+".ts" || b == cleanName+".tsx" || b == "index.ts" || b == "mod.ts" {
				entryFile = b
				break
			}
		}
		if entryFile == "" {
			entryFile = filepath.Base(opts.Srcs[0])
		}
	}

	// Write ts_metadata.json
	var imports map[string]string
	if im != nil {
		imports = im.Imports
	}
	meta := importmap.ModuleMetadata{
		Name:    opts.ModuleName,
		Entry:   entryFile,
		Imports: imports,
		Files:   files,
	}

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(outDir, "ts_metadata.json"), metaData, 0644)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// ExpandCommaSeparated splits comma- or space-separated strings.
func ExpandCommaSeparated(items []string) []string {
	var result []string
	for _, item := range items {
		for _, part := range strings.Fields(strings.ReplaceAll(item, ",", " ")) {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}
