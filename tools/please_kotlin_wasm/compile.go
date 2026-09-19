package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WasmOptions configures a kotlinc-wasm compilation and linking invocation.
type WasmOptions struct {
	KotlincWasm string
	Out         string
	Srcs        []string
	Deps        []string
	Libraries   []string
	Main        string // "noCall" or "call"
	Target      string // "wasm-js" or "wasm-wasi"
	ModuleName  string
	Flags       []string
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func resolveExecutable(configured, fallbackName string) (string, error) {
	if configured != "" {
		if filepath.IsAbs(configured) || fileExists(configured) {
			return configured, nil
		}
		if path, err := exec.LookPath(configured); err == nil {
			return path, nil
		}
		return configured, nil
	}
	if env := os.Getenv("TOOLS_KOTLINC_WASM"); env != "" {
		return resolveExecutable(env, fallbackName)
	}
	if envKotlinc := os.Getenv("TOOLS_KOTLINC"); envKotlinc != "" {
		adjacent := filepath.Join(filepath.Dir(envKotlinc), fallbackName)
		if fileExists(adjacent) {
			return adjacent, nil
		}
	}
	if path, err := exec.LookPath(fallbackName); err == nil {
		return path, nil
	}
	return fallbackName, nil
}

// ResolveKotlincWasm determines the path to the kotlinc-wasm binary.
func ResolveKotlincWasm(configured string) (string, error) {
	return resolveExecutable(configured, "kotlinc-wasm")
}

// FindWasmStdlib searches for kotlin-stdlib-wasm-js.klib or kotlin-stdlib-wasm-wasi.klib.
func FindWasmStdlib(kotlincWasm string, target string) string {
	stdlibName := "kotlin-stdlib-wasm-js.klib"
	if target == "wasm-wasi" {
		stdlibName = "kotlin-stdlib-wasm-wasi.klib"
	}

	var candidates []string
	if kotlincWasm != "" {
		dir := filepath.Dir(kotlincWasm)
		candidates = append(candidates,
			filepath.Join(dir, "..", "lib", stdlibName),
			filepath.Join(dir, "..", "kotlinc", "lib", stdlibName),
			filepath.Join(dir, "lib", stdlibName),
			filepath.Join(dir, "..", "lib", "kotlin-stdlib-wasm-js.klib"),
		)
	}
	if env := os.Getenv("KOTLIN_WASM_STDLIB"); env != "" {
		candidates = append(candidates, env)
	}
	if envKotlinc := os.Getenv("TOOLS_KOTLINC"); envKotlinc != "" {
		dir := filepath.Dir(envKotlinc)
		candidates = append(candidates,
			filepath.Join(dir, "..", "lib", stdlibName),
			filepath.Join(dir, "..", "kotlinc", "lib", stdlibName),
			filepath.Join(dir, "..", "lib", "kotlin-stdlib-wasm-js.klib"),
		)
	}
	candidates = append(candidates,
		"/home/linuxbrew/.linuxbrew/Cellar/kotlin/2.4.10/libexec/lib/kotlin-stdlib-wasm-js.klib",
	)

	for _, c := range candidates {
		clean := filepath.Clean(c)
		if fileExists(clean) {
			return clean
		}
	}
	return ""
}

// DiscoverWasmSources recursively expands source directories and files to collect .kt files.
func DiscoverWasmSources(srcs []string) []string {
	var files []string
	seen := make(map[string]bool)

	addFile := func(p string) {
		clean := filepath.Clean(p)
		if !seen[clean] && strings.HasSuffix(clean, ".kt") && fileExists(clean) {
			seen[clean] = true
			files = append(files, clean)
		}
	}

	for _, src := range srcs {
		src = strings.TrimSpace(src)
		if src == "" || src == "\\" {
			continue
		}
		fi, err := os.Stat(src)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			_ = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
				if err == nil && info != nil && !info.IsDir() {
					addFile(path)
				}
				return nil
			})
		} else {
			addFile(src)
		}
	}
	return files
}

// DiscoverKlibs collects all .klib dependency files from explicit dependencies.
func DiscoverKlibs(deps []string) []string {
	var klibs []string
	seen := make(map[string]bool)

	addKlib := func(p string) {
		clean := filepath.Clean(p)
		if !seen[clean] && strings.HasSuffix(clean, ".klib") && fileExists(clean) {
			seen[clean] = true
			klibs = append(klibs, clean)
		}
	}

	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		if dep == "" || dep == "\\" {
			continue
		}
		fi, err := os.Stat(dep)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			_ = filepath.Walk(dep, func(path string, info os.FileInfo, err error) error {
				if err == nil && info != nil && !info.IsDir() {
					addKlib(path)
				}
				return nil
			})
		} else {
			addKlib(dep)
		}
	}
	return klibs
}

// ExpandCommaSeparated splits comma-separated strings into individual elements.
func ExpandCommaSeparated(items []string) []string {
	var result []string
	for _, item := range items {
		for _, part := range strings.Split(item, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" && trimmed != "\\" {
				result = append(result, trimmed)
			}
		}
	}
	return result
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

func copyDir(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

// CompileWasm executes the two-phase kotlinc-wasm pipeline to build a WebAssembly module.
func CompileWasm(opts WasmOptions) error {
	if opts.Out == "" {
		return fmt.Errorf("output path (--out) is required")
	}

	allSources := DiscoverWasmSources(append(opts.Srcs, opts.Deps...))
	if len(allSources) == 0 {
		return fmt.Errorf("no Kotlin source files (.kt) found in srcs or deps")
	}

	kotlincWasm, err := ResolveKotlincWasm(opts.KotlincWasm)
	if err != nil {
		return fmt.Errorf("failed to resolve kotlinc-wasm: %w", err)
	}

	// Resolve standard library klib
	stdlib := ""
	for _, lib := range opts.Libraries {
		if strings.HasSuffix(lib, ".klib") && fileExists(lib) {
			stdlib = lib
			break
		}
	}
	if stdlib == "" {
		stdlib = FindWasmStdlib(kotlincWasm, opts.Target)
	}
	if stdlib == "" {
		return fmt.Errorf("failed to locate kotlin-stdlib-wasm klib. Specify --libraries or set KOTLIN_WASM_STDLIB")
	}

	// Collect all klibs (stdlib + dependencies)
	klibList := []string{stdlib}
	depKlibs := DiscoverKlibs(opts.Deps)
	for _, klib := range depKlibs {
		if klib != stdlib {
			klibList = append(klibList, klib)
		}
	}
	for _, lib := range opts.Libraries {
		if lib != stdlib && fileExists(lib) {
			klibList = append(klibList, lib)
		}
	}
	librariesArg := strings.Join(klibList, string(os.PathListSeparator))

	// Setup working directory
	tmpDir, err := os.MkdirTemp("", "kt-wasm-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary working directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	moduleName := opts.ModuleName
	if moduleName == "" {
		base := filepath.Base(opts.Out)
		moduleName = strings.TrimSuffix(base, filepath.Ext(base))
	}

	intermediateKlib := filepath.Join(tmpDir, moduleName+".klib")

	// Phase 1: Compile sources to intermediate KLIB
	phase1Args := []string{
		"-libraries", librariesArg,
		"-ir-output-dir", intermediateKlib,
		"-ir-output-name", moduleName,
	}
	if opts.Target != "" {
		phase1Args = append(phase1Args, "-Xwasm-target="+opts.Target)
	}
	phase1Args = append(phase1Args, opts.Flags...)
	phase1Args = append(phase1Args, allSources...)

	cmd1 := exec.Command(kotlincWasm, phase1Args...)
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	if err := cmd1.Run(); err != nil {
		return fmt.Errorf("kotlinc-wasm compilation (phase 1) failed: %w", err)
	}

	// Phase 2: Link intermediate KLIB into WebAssembly binary
	phase2OutDir := filepath.Join(tmpDir, "out")
	if err := os.MkdirAll(phase2OutDir, 0755); err != nil {
		return err
	}

	mainMode := opts.Main
	if mainMode == "" {
		mainMode = "noCall"
	}

	phase2Args := []string{
		"-libraries", librariesArg,
		"-Xinclude=" + intermediateKlib,
		"-ir-output-dir", phase2OutDir,
		"-ir-output-name", moduleName,
		"-main", mainMode,
	}
	if opts.Target != "" {
		phase2Args = append(phase2Args, "-Xwasm-target="+opts.Target)
	}
	phase2Args = append(phase2Args, opts.Flags...)

	cmd2 := exec.Command(kotlincWasm, phase2Args...)
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	if err := cmd2.Run(); err != nil {
		return fmt.Errorf("kotlinc-wasm linking (phase 2) failed: %w", err)
	}

	// Phase 3: Deliver output artifact
	generatedWasm := filepath.Join(phase2OutDir, moduleName+".wasm")
	if !fileExists(generatedWasm) {
		// Look for any generated .wasm file
		entries, _ := os.ReadDir(phase2OutDir)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".wasm") {
				generatedWasm = filepath.Join(phase2OutDir, e.Name())
				break
			}
		}
	}

	if !fileExists(generatedWasm) {
		return fmt.Errorf("linking completed but no .wasm binary was produced in %s", phase2OutDir)
	}

	if strings.HasSuffix(opts.Out, ".wasm") {
		return copyFile(generatedWasm, opts.Out)
	}

	// If destination is a directory, copy all produced files (.wasm, .mjs, etc.)
	if err := os.MkdirAll(opts.Out, 0755); err != nil {
		return err
	}
	return copyDir(phase2OutDir, opts.Out)
}
