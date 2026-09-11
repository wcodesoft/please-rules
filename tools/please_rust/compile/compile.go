package compile

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"tools/please_rust/toolchain"
)

// crateMeta mirrors the JSON produced by the download package's crate_meta.json.
type crateMeta struct {
	LibSrc  string `json:"lib_src"`
	Edition string `json:"edition"`
}

// readCrateMeta reads and JSON-decodes a crate_meta.json file produced by the
// download subcommand.
func readCrateMeta(path string) (*crateMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading crate_meta.json %s: %w", path, err)
	}
	var m crateMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing crate_meta.json %s: %w", path, err)
	}
	return &m, nil
}

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

// sanitizeCrateName converts hyphenated crate names to underscores for rustc extern/crate identification.
func sanitizeCrateName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

var hashSuffixRegex = regexp.MustCompile(`-[0-9a-fA-F]{16}$`)

// extractCrateName derives the crate name from a library artifact filename.
func extractCrateName(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	trimmed := strings.TrimSuffix(base, ext)
	trimmed = strings.TrimPrefix(trimmed, "lib")
	trimmed = hashSuffixRegex.ReplaceAllString(trimmed, "")
	return sanitizeCrateName(trimmed)
}

// findMatchingInput checks if target exists on filesystem or matches an input path.
func findMatchingInput(target string, inputs []string) string {
	if _, err := os.Stat(target); err == nil {
		return target
	}
	targetBase := filepath.Base(target)
	for _, input := range inputs {
		if input == target || filepath.Base(input) == targetBase || strings.HasSuffix(input, target) {
			if _, err := os.Stat(input); err == nil {
				return input
			}
		}
	}
	return ""
}

// findFirstRsFile finds the first existing .rs file from inputs or current directory.
func findFirstRsFile(inputs []string) string {
	for _, input := range inputs {
		if filepath.Ext(input) == ".rs" {
			if _, err := os.Stat(input); err == nil {
				return input
			}
		}
	}

	if entries, err := os.ReadDir("."); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".rs" {
				return e.Name()
			}
		}
	}
	return ""
}

// resolveMainSrc finds the actual entrypoint source file among inputs and current directory.
func resolveMainSrc(mainSrc string, crateType string, inputs []string) string {
	candidates := []string{"lib.rs", "main.rs", "src/lib.rs", "src/main.rs"}
	if crateType == "bin" {
		candidates = []string{"main.rs", "src/main.rs", "lib.rs", "src/lib.rs"}
	}
	if mainSrc != "" {
		candidates = append([]string{mainSrc}, candidates...)
	}

	for _, cand := range candidates {
		if found := findMatchingInput(cand, inputs); found != "" {
			return found
		}
	}

	if fallback := findFirstRsFile(inputs); fallback != "" {
		return fallback
	}

	return mainSrc
}

// isLibFile returns true if the given filename has a shared/static library extension.
func isLibFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".rlib" || ext == ".so" || ext == ".dylib" || ext == ".dll"
}

// discoverDepFiles scans directories in inputs or the current build directory for .rlib and .so files.
func discoverDepFiles(inputs []string) []string {
	seen := make(map[string]bool)
	var deps []string

	addDep := func(p string) {
		clean := filepath.Clean(p)
		if !seen[clean] {
			seen[clean] = true
			deps = append(deps, clean)
		}
	}

	for _, input := range inputs {
		if isLibFile(input) {
			addDep(input)
		}
	}

	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() && isLibFile(path) {
			addDep(path)
		}
		return nil
	})

	return deps
}

// updateExternMap registers depPath under its crate name if not already seen or preferred over hashed names.
func updateExternMap(externMap map[string]string, depPath, selfSanitized string) {
	cName := extractCrateName(depPath)
	if cName == "" || cName == selfSanitized {
		return
	}
	existing, exists := externMap[cName]
	isUnhashed := !strings.Contains(filepath.Base(depPath), "-")
	hasHashedExisting := exists && strings.Contains(filepath.Base(existing), "-")
	if !exists || (isUnhashed && hasHashedExisting) {
		externMap[cName] = depPath
	}
}

// resolveExternFlags builds search directory (-L) and extern library (--extern) arguments.
func resolveExternFlags(depPaths []string, selfCrate string) []string {
	searchDirs := make(map[string]bool)
	externMap := make(map[string]string)
	selfSanitized := sanitizeCrateName(selfCrate)

	for _, depPath := range depPaths {
		if dir := filepath.Dir(depPath); dir != "" && dir != "." {
			searchDirs[dir] = true
		}
		updateExternMap(externMap, depPath, selfSanitized)
	}

	var args []string
	for dir := range searchDirs {
		args = append(args, "-L", fmt.Sprintf("dependency=%s", dir))
	}
	args = append(args, "-L", "dependency=.")

	for cName, path := range externMap {
		args = append(args, "--extern", fmt.Sprintf("%s=%s", cName, path))
	}

	return args
}

// BuildRustcArgs assembles the arguments for invoking rustc.
func BuildRustcArgs(opts Options) ([]string, error) {

	if opts.CrateName == "" {
		return nil, fmt.Errorf("crate name (--crate-name) is required")
	}
	if opts.Out == "" {
		return nil, fmt.Errorf("output path (--out) is required")
	}

	// If a crate_meta.json was provided (from the download rule), use it to
	// override MainSrc and Edition.
	if opts.Meta != "" {
		meta, err := readCrateMeta(opts.Meta)
		if err != nil {
			return nil, err
		}
		if meta.LibSrc != "" {
			opts.MainSrc = meta.LibSrc
		}
		if meta.Edition != "" {
			opts.Edition = meta.Edition
		}
	}

	mainFile := resolveMainSrc(opts.MainSrc, opts.CrateType, opts.Inputs)
	if mainFile == "" {
		return nil, fmt.Errorf("main source file could not be determined")
	}

	args := []string{}
	if opts.Edition != "" {
		args = append(args, "--edition", opts.Edition)
	}

	if opts.CrateType == "test" {
		args = append(args, "--test")
	} else if opts.CrateType != "" {
		args = append(args, "--crate-type", opts.CrateType)
		if opts.CrateType == "proc-macro" {
			args = append(args, "--extern", "proc_macro")
		}
	}

	args = append(args, "--crate-name", sanitizeCrateName(opts.CrateName))

	args = append(args, "-o", opts.Out)

	allDeps := discoverDepFiles(opts.Inputs)
	args = append(args, resolveExternFlags(allDeps, opts.CrateName)...)

	if opts.Flags != "" {
		args = append(args, strings.Fields(opts.Flags)...)
	}

	// Feature flags: --cfg feature="foo"
	for _, feat := range opts.Features {
		args = append(args, "--cfg", fmt.Sprintf("feature=%q", feat))
	}

	// Native static library: -L native=<dir> -l static=<name>
	if opts.NativeLib != "" {
		dir := filepath.Dir(opts.NativeLib)
		if dir == "" {
			dir = "."
		}
		libName := strings.TrimSuffix(filepath.Base(opts.NativeLib), ".a")
		libName = strings.TrimPrefix(libName, "lib")
		args = append(args, "-L", fmt.Sprintf("native=%s", dir))
		args = append(args, "-l", fmt.Sprintf("static=%s", libName))
	}


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

// invokeRustc executes rustc with the provided arguments and environment.
func invokeRustc(rustcPath string, args []string, version string) error {
	cmd := exec.Command(rustcPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if version != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("CARGO_PKG_VERSION=%s", version))
		parts := strings.Split(version, ".")
		if len(parts) >= 1 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("CARGO_PKG_VERSION_MAJOR=%s", parts[0]))
		}
		if len(parts) >= 2 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("CARGO_PKG_VERSION_MINOR=%s", parts[1]))
		}
		if len(parts) >= 3 {
			// Strip any prerelease or build metadata if present
			patch := parts[2]
			if idx := strings.IndexAny(patch, "-+"); idx != -1 {
				patch = patch[:idx]
			}
			cmd.Env = append(cmd.Env, fmt.Sprintf("CARGO_PKG_VERSION_PATCH=%s", patch))
		}
	}


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

