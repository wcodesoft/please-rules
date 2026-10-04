package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"tools/please_ts/importmap"
	"tools/please_ts/npmcache"
)

// esbuildCacheVersions maps a Deno version to the esbuild directory under DENO_DIR/dl in
// which `deno bundle` looks for the esbuild binary before it downloads one. The layout is
// internal to Deno: add an entry when the default Deno version of the toolchain moves (the npm
// bundle fixtures under test/ts/npm_cache fail until it is there).
var esbuildCacheVersions = map[string]string{
	"2.9.7": "0.25.5-1",
}

// denoVersion returns the version of the deno executable.
func denoVersion(denoBin string) (string, error) {
	out, err := exec.Command(denoBin, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("running %s --version: %w", denoBin, err)
	}
	// The first line reads "deno 2.9.7 (stable, release, x86_64-unknown-linux-gnu)".
	first, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	fields := strings.Fields(first)
	if len(fields) < 2 || fields[0] != "deno" {
		return "", fmt.Errorf("cannot read the version from %q", first)
	}
	return fields[1], nil
}

// esbuildPlatform names the platform the way Deno does in its esbuild file name.
func esbuildPlatform(goos, goarch string) (string, error) {
	arch := map[string]string{"amd64": "x64", "arm64": "arm64"}[goarch]
	if (goos != "linux" && goos != "darwin") || arch == "" {
		return "", fmt.Errorf("no esbuild binary for %s/%s", goos, goarch)
	}
	return goos + "-" + arch, nil
}

// placeEsbuild makes `deno bundle` find the pinned esbuild in denoDir instead of
// downloading its own: Deno looks for dl/esbuild-<version>/esbuild-<os>-<arch>.
func placeEsbuild(denoBin, denoDir string, opts Options) error {
	if opts.EsbuildBinary == "" {
		return fmt.Errorf("bundling npm packages needs the pinned esbuild binary (the esbuild_tool config, //tools/esbuild_toolchain by default); none was given")
	}
	version := opts.EsbuildCacheVersion
	if version == "" {
		deno, err := denoVersion(denoBin)
		if err != nil {
			return err
		}
		var ok bool
		if version, ok = esbuildCacheVersions[deno]; !ok {
			return fmt.Errorf("this tool does not know where Deno %s looks for esbuild; pass --esbuild-cache-version (the <version> of DENO_DIR/dl/esbuild-<version>)", deno)
		}
	}
	platform, err := esbuildPlatform(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	src, err := filepath.Abs(opts.EsbuildBinary)
	if err != nil {
		return err
	}
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("esbuild binary: %w", err)
	}
	dir := filepath.Join(denoDir, "dl", "esbuild-"+version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.Symlink(src, filepath.Join(dir, "esbuild-"+platform))
}

// runDenoBundle bundles with `deno bundle`, which resolves the npm: specifiers of the import
// map from the merged cache slices (offline: no node_modules, no lockfile, nothing fetched)
// and runs the pinned esbuild.
func runDenoBundle(opts Options, im *importmap.ImportMap) error {
	denoBin := opts.Deno
	if denoBin == "" {
		denoBin = "deno"
	}

	// The bundle names the files it was built from, relative to the working directory, so the
	// per-run DENO_DIR is a fixed directory there and not a random one under the temp
	// directory: the same sources must give the same bundle.
	denoDir, err := filepath.Abs(".please_ts_deno")
	if err != nil {
		return err
	}
	if err := os.RemoveAll(denoDir); err != nil {
		return err
	}
	if err := os.MkdirAll(denoDir, 0755); err != nil {
		return fmt.Errorf("failed to create DENO_DIR: %w", err)
	}
	defer os.RemoveAll(denoDir)
	if _, err := npmcache.Prepare(".", denoDir); err != nil {
		return fmt.Errorf("failed preparing the npm cache: %w", err)
	}
	if err := placeEsbuild(denoBin, denoDir, opts); err != nil {
		return err
	}

	importMapPath := ".import_map.json"
	if err := im.WriteToFile(importMapPath); err != nil {
		return fmt.Errorf("failed writing import map: %w", err)
	}
	defer os.Remove(importMapPath)

	args := []string{"bundle", "--no-remote", "--import-map", importMapPath, "--platform", "browser", "-o", opts.Out}
	if opts.Format != "" {
		args = append(args, "--format", opts.Format)
	}
	if opts.Minify {
		args = append(args, "--minify")
	}
	if opts.Sourcemap {
		args = append(args, "--sourcemap")
	}
	args = append(args, opts.Flags...)
	args = append(args, opts.Main)

	cmd := exec.Command(denoBin, args...)
	cmd.Env = append(os.Environ(), "DENO_DIR="+denoDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("deno bundle failed: %w", err)
	}

	// Deno fetches its own esbuild from the registry when it does not find the pinned one
	// where it looks, and --no-remote does not stop that. Do not let that pass silently.
	if _, err := os.Stat(filepath.Join(denoDir, "npm", "registry.npmjs.org", "@esbuild")); err == nil {
		return fmt.Errorf("deno bundle downloaded esbuild instead of using the pinned binary: its lookup path for this Deno version differs from the one this tool uses (--esbuild-cache-version)")
	}
	return nil
}
