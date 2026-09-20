package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"tools/please_ts/importmap"
)

// Options holds configuration for bundling JavaScript/TypeScript assets.
type Options struct {
	Deno        string
	BundlerTool string
	Out         string
	Main        string
	Srcs        []string
	Deps        []string
	ModuleName  string
	Format      string // "esm" | "iife"
	Minify      bool
	Sourcemap   bool
	Flags       []string
}

// Run bundles the source files into a single distribution asset using esbuild by default.
func Run(opts Options) error {
	if opts.Main == "" {
		return fmt.Errorf("main entry point must be specified for ts_bundle")
	}
	if opts.Out == "" {
		return fmt.Errorf("output bundle path must be specified for ts_bundle")
	}

	if err := os.MkdirAll(filepath.Dir(opts.Out), 0755); err != nil {
		return err
	}

	return runEsbuildBundler(opts)
}

func buildEsbuildArgs(opts Options, im *importmap.ImportMap) []string {
	args := []string{opts.Main, "--outfile=" + opts.Out, "--bundle"}
	if opts.Format != "" {
		args = append(args, "--format="+opts.Format)
	}
	if opts.Minify {
		args = append(args, "--minify")
	}
	if opts.Sourcemap {
		args = append(args, "--sourcemap")
	}

	if im != nil {
		keys := make([]string, 0, len(im.Imports))
		for k := range im.Imports {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if len(keys[i]) != len(keys[j]) {
				return len(keys[i]) > len(keys[j])
			}
			return keys[i] < keys[j]
		})

		for _, k := range keys {
			target := im.Imports[k]
			kClean := strings.TrimSuffix(k, "/")
			tClean := strings.TrimSuffix(target, "/")
			if kClean != "" && tClean != "" {
				args = append(args, fmt.Sprintf("--alias:%s=%s", kClean, tClean))
			}
		}
	}

	args = append(args, opts.Flags...)
	return args
}

func runEsbuildBundler(opts Options) error {
	im, err := importmap.Synthesize(opts.ModuleName, opts.Srcs, opts.Deps, ".")
	if err != nil {
		return fmt.Errorf("failed synthesizing import map: %w", err)
	}

	args := buildEsbuildArgs(opts, im)

	// Execute bundler:
	// If an external bundler tool is explicitly configured, use it.
	// Otherwise, use the hermetic Deno executable (passed via opts.Deno) to execute esbuild.
	var cmd *exec.Cmd
	if opts.BundlerTool != "" {
		cmd = exec.Command(opts.BundlerTool, args...)
	} else {
		denoBin := opts.Deno
		if denoBin == "" {
			denoBin = "deno"
		}
		denoArgs := append([]string{"run", "-A", "npm:esbuild"}, args...)
		cmd = exec.Command(denoBin, denoArgs...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
