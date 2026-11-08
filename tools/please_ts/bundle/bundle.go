package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"tools/please_ts/css"
	"tools/please_ts/importmap"
)

// Options holds configuration for bundling JavaScript/TypeScript and CSS assets.
type Options struct {
	Deno        string
	BundlerTool string
	Out         string
	CSSOut      string
	Main        string
	Srcs        []string
	Deps        []string
	ModuleName  string
	Format      string // "esm" | "iife"
	Minify      bool
	Sourcemap   bool
	Flags       []string

	// AllowCSSOverrides disables strict cross-file CSS selector/variable collision checking.
	AllowCSSOverrides bool

	// EsbuildBinary is the pinned esbuild executable that `deno bundle` and esbuild bundler run.
	EsbuildBinary string
	// EsbuildCacheVersion overrides the directory (esbuild-<version>) under DENO_DIR/dl in
	// which that Deno version looks for its esbuild, for a Deno version this tool does not know.
	EsbuildCacheVersion string
}

// Run bundles the source files into a single distribution asset using esbuild by default.
func Run(opts Options) error {
	if opts.Out == "" {
		return fmt.Errorf("output bundle path must be specified for ts_bundle")
	}

	isPureCSS := css.IsCSSFile(opts.Out) || css.IsCSSFile(opts.Main) || (opts.Main == "" && hasAnyCSSSources(opts.Srcs))
	if !isPureCSS && opts.Main == "" {
		return fmt.Errorf("main entry point must be specified for ts_bundle")
	}

	if err := os.MkdirAll(filepath.Dir(opts.Out), 0755); err != nil {
		return err
	}

	im, err := importmap.Synthesize(opts.ModuleName, opts.Srcs, opts.Deps, ".")
	if err != nil {
		return fmt.Errorf("failed synthesizing import map: %w", err)
	}

	if isPureCSS {
		return bundleCSSOutput(opts, opts.Out, im)
	}

	// Bundle JS/TS output first
	if im.HasNpmSpecifiers() {
		if err := runDenoBundle(opts, im); err != nil {
			return err
		}
	} else {
		if err := runEsbuildBundler(opts, im); err != nil {
			return err
		}
	}

	// If css_out is also requested on a TS bundle, collect and bundle all transitive CSS files.
	if opts.CSSOut != "" {
		if err := os.MkdirAll(filepath.Dir(opts.CSSOut), 0755); err != nil {
			return err
		}
		cssOpts := opts
		cssOpts.Main = ""
		if err := bundleCSSOutput(cssOpts, opts.CSSOut, im); err != nil {
			return err
		}
	}

	return nil
}

func hasAnyCSSSources(srcs []string) bool {
	for _, s := range srcs {
		if css.IsCSSFile(s) {
			return true
		}
	}
	return false
}

func bundleCSSOutput(opts Options, outPath string, im *importmap.ImportMap) error {
	graphCSS, err := css.DiscoverGraphFiles(opts.Deps, opts.Srcs, ".")
	if err != nil {
		return fmt.Errorf("discovering CSS dependencies: %w", err)
	}
	if opts.Main != "" && css.IsCSSFile(opts.Main) {
		found := false
		cleanMain := filepath.Clean(opts.Main)
		for _, f := range graphCSS {
			if filepath.Clean(f) == cleanMain {
				found = true
				break
			}
		}
		if !found {
			graphCSS = append(graphCSS, cleanMain)
		}
	}
	if len(graphCSS) == 0 {
		return fmt.Errorf("no CSS files found in srcs or deps for CSS bundle %s", outPath)
	}

	if !opts.AllowCSSOverrides {
		if err := css.CheckFiles(graphCSS); err != nil {
			return err
		}
	}

	// Synthesize a temporary CSS entry point that @imports all graph CSS files in topological order.
	// esbuild deduplicates any overlapping @import statements inside individual files automatically.
	entryFile, cleanup, err := writeCSSEntryFile(opts, graphCSS)
	if err != nil {
		return err
	}
	defer cleanup()

	cssOpts := opts
	cssOpts.Main = entryFile
	cssOpts.Out = outPath
	cssOpts.Format = "" // --format is not applicable to CSS-only bundles in esbuild
	return runEsbuildBundler(cssOpts, im)
}

func writeCSSEntryFile(opts Options, graphCSS []string) (string, func(), error) {
	// If there is only a single CSS file (and no transitive dependency CSS files), pass it directly.
	if len(graphCSS) == 1 && opts.Main != "" && filepath.Clean(opts.Main) == filepath.Clean(graphCSS[0]) {
		return opts.Main, func() {}, nil
	}

	var sb strings.Builder
	for _, f := range graphCSS {
		rel := f
		if !strings.HasPrefix(rel, "./") && !strings.HasPrefix(rel, "../") && !filepath.IsAbs(rel) {
			rel = "./" + filepath.ToSlash(rel)
		} else {
			rel = filepath.ToSlash(rel)
		}
		sb.WriteString(fmt.Sprintf("@import %q;\n", rel))
	}

	tmpFile, err := os.CreateTemp(".", ".please_ts_css_entry_*.css")
	if err != nil {
		return "", nil, fmt.Errorf("creating temporary CSS entry: %w", err)
	}
	name := tmpFile.Name()
	if _, err := tmpFile.WriteString(sb.String()); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(name)
		return "", nil, err
	}
	_ = tmpFile.Close()
	return name, func() { _ = os.Remove(name) }, nil
}

func isValidEsbuildAlias(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return false
	}
	if strings.Contains(name, "://") {
		return false
	}
	return true
}

func buildEsbuildArgs(opts Options, im *importmap.ImportMap) []string {
	args := []string{opts.Main, "--outfile=" + opts.Out, "--bundle"}
	if opts.Format != "" && !css.IsCSSFile(opts.Out) && !css.IsCSSFile(opts.Main) {
		args = append(args, "--format="+opts.Format)
	}
	if opts.Minify {
		args = append(args, "--minify")
	}
	if opts.Sourcemap {
		args = append(args, "--sourcemap")
	}

	if im != nil {
		aliasMap := make(map[string]string)
		// First pass: add exact matches (keys without trailing slash), which typically point to entry files.
		for k, target := range im.Imports {
			if strings.HasSuffix(k, "/") {
				continue
			}
			if !isValidEsbuildAlias(k) {
				continue
			}
			tClean := strings.TrimSuffix(target, "/")
			if tClean != "" {
				aliasMap[k] = tClean
			}
		}
		// Second pass: add directory prefix matches if not already mapped.
		for k, target := range im.Imports {
			if !strings.HasSuffix(k, "/") {
				continue
			}
			kClean := strings.TrimSuffix(k, "/")
			if !isValidEsbuildAlias(kClean) {
				continue
			}
			if _, exists := aliasMap[kClean]; !exists {
				tClean := strings.TrimSuffix(target, "/")
				if tClean != "" {
					aliasMap[kClean] = tClean
				}
			}
		}

		aliasKeys := make([]string, 0, len(aliasMap))
		for k := range aliasMap {
			aliasKeys = append(aliasKeys, k)
		}
		sort.Strings(aliasKeys)

		for _, k := range aliasKeys {
			args = append(args, fmt.Sprintf("--alias:%s=%s", k, aliasMap[k]))
		}
	}

	args = append(args, opts.Flags...)
	return args
}

func runEsbuildBundler(opts Options, im *importmap.ImportMap) error {
	args := buildEsbuildArgs(opts, im)

	// Execute bundler:
	// 1. If an external bundler tool is explicitly configured, use it.
	// 2. If the pinned EsbuildBinary is provided (from //tools/esbuild_toolchain), invoke it directly (100% offline).
	// 3. Otherwise, fall back to Deno running npm:esbuild.
	var cmd *exec.Cmd
	switch {
	case opts.BundlerTool != "":
		cmd = exec.Command(opts.BundlerTool, args...)
	case opts.EsbuildBinary != "":
		cmd = exec.Command(opts.EsbuildBinary, args...)
	default:
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
