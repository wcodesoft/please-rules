package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	Wit         string // Path to .wit file, directory, or WIT target
	Impl        string // Optional implementation class override
	Flags       []string
}

// WitFunc represents a function signature parsed from WIT.
type WitFunc struct {
	Name       string
	Params     []WitParam
	ReturnType string
}

// WitParam represents a function parameter in WIT.
type WitParam struct {
	Name string
	Type string
}

// WitInterface represents an interface parsed from WIT.
type WitInterface struct {
	Name      string
	Package   string
	Functions []WitFunc
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

	for _, c := range candidates {
		clean := filepath.Clean(c)
		if fileExists(clean) {
			return clean
		}
	}
	return ""
}

// ToCamelCase converts kebab-case or snake_case identifiers to camelCase.
func ToCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(parts) == 0 {
		return s
	}
	res := strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			res += strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	return res
}

// ToPascalCase converts kebab-case or snake_case identifiers to PascalCase.
func ToPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(parts) == 0 {
		return s
	}
	var res string
	for _, p := range parts {
		if len(p) > 0 {
			res += strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return res
}

// MapWitTypeToKotlin maps standard WIT primitives to Kotlin types.
func MapWitTypeToKotlin(witType string) string {
	witType = strings.TrimSpace(witType)
	switch witType {
	case "s8":
		return "Byte"
	case "s16":
		return "Short"
	case "s32", "u8", "u16", "u32":
		return "Int"
	case "s64", "u64":
		return "Long"
	case "f32":
		return "Float"
	case "f64":
		return "Double"
	case "bool":
		return "Boolean"
	case "string":
		return "String"
	case "_", "unit":
		return "Unit"
	}

	// Handle result<T, E>
	if strings.HasPrefix(witType, "result<") && strings.HasSuffix(witType, ">") {
		inner := witType[7 : len(witType)-1]
		parts := strings.Split(inner, ",")
		if len(parts) > 0 {
			okType := strings.TrimSpace(parts[0])
			if okType == "_" {
				return "Unit"
			}
			mapped := MapWitTypeToKotlin(okType)
			if mapped == "Unit" {
				return "Unit"
			}
			return mapped + "?"
		}
		return "Unit"
	}

	// Handle list<T>
	if strings.HasPrefix(witType, "list<") && strings.HasSuffix(witType, ">") {
		elemType := witType[5 : len(witType)-1]
		return fmt.Sprintf("List<%s>", MapWitTypeToKotlin(elemType))
	}

	// Handle option<T>
	if strings.HasPrefix(witType, "option<") && strings.HasSuffix(witType, ">") {
		elemType := witType[7 : len(witType)-1]
		return MapWitTypeToKotlin(elemType) + "?"
	}

	return ToPascalCase(witType)
}

var funcRegex = regexp.MustCompile(`^\s*([a-zA-Z0-9_-]+)\s*:\s*func\s*\((.*?)\)(?:\s*->\s*(.+))?`)
var packageRegex = regexp.MustCompile(`(?m)^\s*package\s+([a-zA-Z0-9_:-]+);`)
var ifaceRegex = regexp.MustCompile(`^\s*interface\s+([a-zA-Z0-9_-]+)\s*\{`)
var resourceRegex = regexp.MustCompile(`^\s*resource\s+([a-zA-Z0-9_-]+)\s*\{`)

// ParseWit parses .wit files in a given path into WitInterface structures.
func ParseWit(witPath string) ([]WitInterface, error) {
	var files []string
	fi, err := os.Stat(witPath)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		_ = filepath.Walk(witPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && info != nil && !info.IsDir() && strings.HasSuffix(path, ".wit") {
				files = append(files, path)
			}
			return nil
		})
	} else {
		files = append(files, witPath)
	}

	var interfaces []WitInterface

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		pkgName := ""
		if match := packageRegex.FindStringSubmatch(string(data)); len(match) > 1 {
			pkgName = strings.ReplaceAll(match[1], ":", ".")
		}

		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		var currentIface *WitInterface

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "//") {
				continue
			}

			if m := ifaceRegex.FindStringSubmatch(line); len(m) > 1 {
				interfaces = append(interfaces, WitInterface{
					Name:    m[1],
					Package: pkgName,
				})
				currentIface = &interfaces[len(interfaces)-1]
				continue
			}

			if currentIface != nil {
				if m := resourceRegex.FindStringSubmatch(line); len(m) > 1 {
					continue
				}

				if m := funcRegex.FindStringSubmatch(line); len(m) > 1 {
					funcName := m[1]
					rawParams := m[2]
					rawReturn := strings.TrimSuffix(strings.TrimSpace(m[3]), ";")

					var params []WitParam
					if rawParams != "" {
						for _, p := range strings.Split(rawParams, ",") {
							p = strings.TrimSpace(p)
							if p == "" {
								continue
							}
							parts := strings.SplitN(p, ":", 2)
							if len(parts) == 2 {
								pName := ToCamelCase(strings.TrimSpace(parts[0]))
								pType := MapWitTypeToKotlin(strings.TrimSpace(parts[1]))
								params = append(params, WitParam{Name: pName, Type: pType})
							}
						}
					}

					retType := "Unit"
					if rawReturn != "" {
						retType = MapWitTypeToKotlin(rawReturn)
					}

					currentIface.Functions = append(currentIface.Functions, WitFunc{
						Name:       funcName,
						Params:     params,
						ReturnType: retType,
					})
				}
			}
		}
	}

	return interfaces, nil
}

// DetectPackage finds the declared Kotlin package across source files.
func DetectPackage(srcs []string) string {
	pkgRegex := regexp.MustCompile(`(?m)^\s*package\s+([a-zA-Z0-9_.]+)`)
	for _, src := range srcs {
		data, err := os.ReadFile(src)
		if err == nil {
			if m := pkgRegex.FindStringSubmatch(string(data)); len(m) > 1 {
				return m[1]
			}
		}
	}
	return ""
}

func hasClassInSources(sources []string, className string) bool {
	classRegex := regexp.MustCompile(`(?m)\bclass\s+` + regexp.QuoteMeta(className) + `\b`)
	for _, src := range sources {
		data, err := os.ReadFile(src)
		if err == nil {
			if classRegex.Match(data) {
				return true
			}
		}
	}
	return false
}

// GenerateWitArtifacts generates the Kotlin interface and @WasmExport bridge from WIT.
func GenerateWitArtifacts(interfaces []WitInterface, implClass, targetPkg, outDir string, sources ...string) ([]string, error) {
	var generated []string

	for _, iface := range interfaces {
		pascalName := ToPascalCase(iface.Name)
		resolvedImpl := implClass
		if resolvedImpl == "" {
			if hasClassInSources(sources, pascalName) {
				resolvedImpl = pascalName
			} else {
				resolvedImpl = pascalName + "Impl"
			}
		}

		ifacePkg := iface.Package
		if ifacePkg == "" {
			ifacePkg = targetPkg
		}
		if ifacePkg == "" {
			ifacePkg = "wit.generated"
		}

		bridgePkg := targetPkg
		if bridgePkg == "" {
			bridgePkg = ifacePkg
		}

		// 1. Generate Interface (<Interface>.kt)
		var ifaceSb strings.Builder
		ifaceSb.WriteString(fmt.Sprintf("// Auto-generated interface from WIT. DO NOT EDIT.\npackage %s\n\n", ifacePkg))
		ifaceSb.WriteString(fmt.Sprintf("public interface %s {\n", pascalName))

		for _, fn := range iface.Functions {
			camelName := ToCamelCase(fn.Name)
			var params []string
			for _, p := range fn.Params {
				params = append(params, fmt.Sprintf("%s: %s", p.Name, p.Type))
			}
			retClause := ""
			if fn.ReturnType != "" && fn.ReturnType != "Unit" {
				retClause = ": " + fn.ReturnType
			}
			ifaceSb.WriteString(fmt.Sprintf("    fun %s(%s)%s\n", camelName, strings.Join(params, ", "), retClause))
		}
		ifaceSb.WriteString("}\n")

		ifaceFile := filepath.Join(outDir, pascalName+".kt")
		if err := os.WriteFile(ifaceFile, []byte(ifaceSb.String()), 0644); err != nil {
			return nil, err
		}
		generated = append(generated, ifaceFile)

		// 2. Generate Export Bridge (<Interface>Bridge.kt)
		var bridgeSb strings.Builder
		bridgeSb.WriteString(fmt.Sprintf("// Auto-generated WebAssembly export bridge from WIT. DO NOT EDIT.\npackage %s\n\n", bridgePkg))
		bridgeSb.WriteString("import kotlin.wasm.WasmExport\n")
		if bridgePkg != ifacePkg && ifacePkg != "" {
			bridgeSb.WriteString(fmt.Sprintf("import %s.*\n", ifacePkg))
		}
		bridgeSb.WriteString("\n")
		bridgeSb.WriteString(fmt.Sprintf("private val instance by lazy { %s() }\n\n", resolvedImpl))

		for _, fn := range iface.Functions {
			camelName := ToCamelCase(fn.Name)
			var params []string
			var callArgs []string
			for _, p := range fn.Params {
				params = append(params, fmt.Sprintf("%s: %s", p.Name, p.Type))
				callArgs = append(callArgs, p.Name)
			}

			bridgeSb.WriteString("@WasmExport\n")
			callExpr := fmt.Sprintf("instance.%s(%s)", camelName, strings.Join(callArgs, ", "))

			if fn.ReturnType == "" || fn.ReturnType == "Unit" {
				bridgeSb.WriteString(fmt.Sprintf("fun %s(%s) {\n    %s\n}\n\n", camelName, strings.Join(params, ", "), callExpr))
			} else if strings.HasSuffix(fn.ReturnType, "?") {
				// Provide fallback for nullable returns to Wasm
				baseType := strings.TrimSuffix(fn.ReturnType, "?")
				fallback := "-1"
				if baseType == "Boolean" {
					fallback = "false"
				} else if baseType == "String" {
					fallback = `""`
				}
				bridgeSb.WriteString(fmt.Sprintf("fun %s(%s): %s {\n    return %s ?: %s\n}\n\n", camelName, strings.Join(params, ", "), baseType, callExpr, fallback))
			} else {
				bridgeSb.WriteString(fmt.Sprintf("fun %s(%s): %s {\n    return %s\n}\n\n", camelName, strings.Join(params, ", "), fn.ReturnType, callExpr))
			}
		}

		bridgeFile := filepath.Join(outDir, pascalName+"Bridge.kt")
		if err := os.WriteFile(bridgeFile, []byte(bridgeSb.String()), 0644); err != nil {
			return nil, err
		}
		generated = append(generated, bridgeFile)
	}

	return generated, nil
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

// ExpandCommaSeparated splits comma- or whitespace-separated strings into individual elements.
func ExpandCommaSeparated(items []string) []string {
	var result []string
	for _, item := range items {
		normalized := strings.ReplaceAll(item, ",", " ")
		for _, part := range strings.Fields(normalized) {
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

	// Setup working directory
	tmpDir, err := os.MkdirTemp("", "kt-wasm-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary working directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	allSources := DiscoverWasmSources(append(opts.Srcs, opts.Deps...))

	// If WIT contract is specified, parse it and auto-generate interface & bridge
	if opts.Wit != "" {
		witInterfaces, err := ParseWit(opts.Wit)
		if err != nil {
			return fmt.Errorf("failed to parse WIT definitions from %s: %w", opts.Wit, err)
		}
		if len(witInterfaces) > 0 {
			targetPkg := DetectPackage(allSources)
			generatedFiles, err := GenerateWitArtifacts(witInterfaces, opts.Impl, targetPkg, tmpDir, allSources...)
			if err != nil {
				return fmt.Errorf("failed to generate WIT artifacts: %w", err)
			}
			allSources = append(allSources, generatedFiles...)
		}
	}

	if len(allSources) == 0 {
		return fmt.Errorf("no Kotlin source files (.kt) found in srcs, deps, or generated from wit")
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
	depKlibs := DiscoverKlibs(opts.Deps)
	if stdlib == "" {
		stdlib = FindWasmStdlib(kotlincWasm, opts.Target)
	}
	if stdlib == "" {
		stdlibName := "kotlin-stdlib-wasm-js.klib"
		if opts.Target == "wasm-wasi" {
			stdlibName = "kotlin-stdlib-wasm-wasi.klib"
		}
		for _, dep := range depKlibs {
			if strings.Contains(filepath.Base(dep), stdlibName) {
				stdlib = dep
				break
			}
		}
	}
	if stdlib == "" {
		return fmt.Errorf("failed to locate kotlin-stdlib-wasm klib. Specify --libraries or set KOTLIN_WASM_STDLIB")
	}

	// Collect all klibs (stdlib + dependencies)
	klibList := []string{stdlib}
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
