package binary

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"tools/please_ts/importmap"
)

// Options holds configuration for compiling a standalone executable via deno compile.
type Options struct {
	Deno       string
	Out        string
	Main       string
	Srcs       []string
	Deps       []string
	ModuleName string
	Flags      []string
}

// Run compiles the TypeScript program into a standalone executable.
func Run(opts Options) error {
	if opts.Main == "" {
		return fmt.Errorf("main entry point must be specified for ts_binary")
	}
	if opts.Out == "" {
		return fmt.Errorf("output binary path must be specified for ts_binary")
	}

	denoBin := opts.Deno
	if denoBin == "" {
		denoBin = "deno"
	}

	tmpDir, err := os.MkdirTemp("", "please_ts_binary_*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	denoCacheDir := filepath.Join(tmpDir, ".deno_cache")
	if err := os.MkdirAll(denoCacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create DENO_DIR: %w", err)
	}

	// Synthesize import map
	importMapPath := ".import_map.json"
	im, err := importmap.Synthesize(opts.ModuleName, opts.Srcs, opts.Deps, ".")
	if err != nil {
		return fmt.Errorf("failed synthesizing import map: %w", err)
	}
	if err := im.WriteToFile(importMapPath); err != nil {
		return fmt.Errorf("failed writing import map: %w", err)
	}
	defer os.Remove(importMapPath)

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(opts.Out), 0755); err != nil {
		return err
	}

	// Run `deno compile --no-remote --import-map ... -o <out> <flags...> <main>`
	args := []string{
		"compile",
		"--no-remote",
		"--import-map", importMapPath,
		"-o", opts.Out,
	}
	args = append(args, opts.Flags...)
	args = append(args, opts.Main)

	cmd := exec.Command(denoBin, args...)
	cmd.Env = append(os.Environ(), "DENO_DIR="+denoCacheDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("deno compile failed: %w", err)
	}

	return nil
}
