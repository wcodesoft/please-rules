package compile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tools/please_rust/toolchain"
)

// Options specifies the parameters needed to compile a Rust target.
type Options struct {
	Out       string
	CrateName string
	CrateType string
	Edition   string
	MainSrc   string
	Version   string
	Flags     string
	Rustc     string
	Inputs    []string
	Meta      string   // path to crate_meta.json (from download rule)
	NativeLib string   // path to a .a static lib to link
	Features  []string // feature flags; each becomes --cfg feature="foo"
}

// buildCrateTypeArgs returns the compiler arguments corresponding to crateType.
func buildCrateTypeArgs(crateType string) []string {
	if crateType == "test" {
		return []string{"--test"}
	}
	if crateType == "" {
		return nil
	}
	args := []string{"--crate-type", crateType}
	if crateType == "proc-macro" {
		args = append(args, "--extern", "proc_macro")
	}
	return args
}

// buildFeatureArgs converts crate feature names to rustc --cfg feature="..." flags.
func buildFeatureArgs(features []string) []string {
	var args []string
	for _, feat := range features {
		args = append(args, "--cfg", fmt.Sprintf("feature=%q", feat))
	}
	return args
}

// buildNativeLibArgs generates search directory (-L) and static link (-l) flags for a native library.
func buildNativeLibArgs(nativeLib string) []string {
	if nativeLib == "" {
		return nil
	}
	dir := filepath.Dir(nativeLib)
	if dir == "" {
		dir = "."
	}
	libName := strings.TrimSuffix(filepath.Base(nativeLib), ".a")
	libName = strings.TrimPrefix(libName, "lib")
	return []string{
		"-L", fmt.Sprintf("native=%s", dir),
		"-l", fmt.Sprintf("static=%s", libName),
	}
}

// BuildRustcArgs assembles the arguments for invoking rustc.
func BuildRustcArgs(opts Options) ([]string, error) {
	if opts.CrateName == "" {
		return nil, fmt.Errorf("crate name (--crate-name) is required")
	}
	if opts.Out == "" {
		return nil, fmt.Errorf("output path (--out) is required")
	}

	if err := applyCrateMeta(&opts); err != nil {
		return nil, err
	}

	mainFile := resolveMainSrc(opts.MainSrc, opts.CrateType, opts.Inputs)
	if mainFile == "" {
		return nil, fmt.Errorf("main source file could not be determined")
	}

	var args []string
	if opts.Edition != "" {
		args = append(args, "--edition", opts.Edition)
	}

	args = append(args, buildCrateTypeArgs(opts.CrateType)...)
	args = append(args, "--crate-name", sanitizeCrateName(opts.CrateName))
	args = append(args, "-o", opts.Out)

	allDeps := discoverDepFiles(opts.Inputs)
	args = append(args, resolveExternFlags(allDeps, opts.CrateName)...)

	if opts.Flags != "" {
		args = append(args, strings.Fields(opts.Flags)...)
	}

	args = append(args, buildFeatureArgs(opts.Features)...)
	args = append(args, buildNativeLibArgs(opts.NativeLib)...)
	args = append(args, mainFile)
	return args, nil
}

// ensureOutputDir creates the parent directory of path if needed.
func ensureOutputDir(path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output dir %s: %w", dir, err)
		}
	}
	return nil
}

// buildCargoVersionEnv constructs CARGO_PKG_VERSION environment variables from a version string.
func buildCargoVersionEnv(version string) []string {
	if version == "" {
		return nil
	}
	env := []string{fmt.Sprintf("CARGO_PKG_VERSION=%s", version)}
	parts := strings.Split(version, ".")
	if len(parts) >= 1 {
		env = append(env, fmt.Sprintf("CARGO_PKG_VERSION_MAJOR=%s", parts[0]))
	}
	if len(parts) >= 2 {
		env = append(env, fmt.Sprintf("CARGO_PKG_VERSION_MINOR=%s", parts[1]))
	}
	if len(parts) >= 3 {
		patch := parts[2]
		if idx := strings.IndexAny(patch, "-+"); idx != -1 {
			patch = patch[:idx]
		}
		env = append(env, fmt.Sprintf("CARGO_PKG_VERSION_PATCH=%s", patch))
	}
	return env
}

// invokeRustc executes rustc with the provided arguments and environment.
func invokeRustc(rustcPath string, args []string, version string) error {
	cmd := exec.Command(rustcPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	cmd.Env = append(cmd.Env, buildCargoVersionEnv(version)...)

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG: rustc command failed: %s %s\n", rustcPath, strings.Join(args, " "))
		return fmt.Errorf("rustc compilation failed: %w", err)
	}
	return nil
}

// Run executes the Rust compilation with rustc.
func Run(opts Options) error {
	rustcPath, err := toolchain.FindRustc(opts.Rustc)
	if err != nil {
		return err
	}

	if err := ensureOutputDir(opts.Out); err != nil {
		return err
	}

	args, err := BuildRustcArgs(opts)
	if err != nil {
		return err
	}

	return invokeRustc(rustcPath, args, opts.Version)
}
