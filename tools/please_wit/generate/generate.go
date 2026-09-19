package generate

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Options struct {
	Bindgen           string
	Lang              string
	Out               string
	Srcs              []string
	Worlds            []string
	Flags             []string
	Package           string
	CompanionFilename string
	ModuleName        string
}

var worldRegex = regexp.MustCompile(`^\s*world\s+([a-zA-Z0-9_-]+)`)
var packageRegex = regexp.MustCompile(`(?m)^\s*package\s+([a-zA-Z0-9_:-]+);`)

// DiscoverWorlds scans all .wit files in a directory or file list and returns declared world names.
func DiscoverWorlds(witPath string) ([]string, error) {
	var files []string
	fi, err := os.Stat(witPath)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		entries, err := os.ReadDir(witPath)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".wit") {
				files = append(files, filepath.Join(witPath, e.Name()))
			}
		}
	} else {
		files = append(files, witPath)
	}

	var worlds []string
	seen := make(map[string]bool)

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			matches := worldRegex.FindStringSubmatch(scanner.Text())
			if len(matches) > 1 {
				name := matches[1]
				if !seen[name] {
					seen[name] = true
					worlds = append(worlds, name)
				}
			}
		}
		f.Close()
	}

	return worlds, nil
}

// DiscoverPackage scans all .wit files in a directory or file list and returns the declared package name if found.
func DiscoverPackage(witPath string) (string, error) {
	var files []string
	fi, err := os.Stat(witPath)
	if err != nil {
		return "", err
	}

	if fi.IsDir() {
		entries, err := os.ReadDir(witPath)
		if err != nil {
			return "", err
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".wit") {
				files = append(files, filepath.Join(witPath, e.Name()))
			}
		}
	} else {
		files = append(files, witPath)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		matches := packageRegex.FindSubmatch(data)
		if len(matches) > 1 {
			return string(matches[1]), nil
		}
	}

	return "", nil
}

// ToPascalCase converts names like "structures" or "two_sum" or "two-sum" to "Structures" or "TwoSum".
func ToPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ':' || r == '.'
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// DeriveBaseName determines the logical base name for outputs based on companion filename, package, worlds, or directory name.
func DeriveBaseName(opts Options, witPath string) string {
	if opts.CompanionFilename != "" {
		base := filepath.Base(opts.CompanionFilename)
		ext := filepath.Ext(base)
		return strings.TrimSuffix(base, ext)
	}

	pkg := opts.Package
	if pkg == "" && witPath != "" {
		pkg, _ = DiscoverPackage(witPath)
	}
	if pkg != "" {
		if idx := strings.LastIndex(pkg, ":"); idx != -1 {
			return pkg[idx+1:]
		}
		return pkg
	}

	if witPath != "" {
		worlds, _ := DiscoverWorlds(witPath)
		if len(worlds) > 0 {
			w := worlds[0]
			w = strings.TrimSuffix(w, "-world")
			w = strings.TrimSuffix(w, "_world")
			return w
		}

		base := filepath.Base(witPath)
		base = strings.TrimSuffix(base, "_wit")
		return base
	}

	return "Wit"
}

// GeneratorForLang maps our supported language string to wit-bindgen generator subcommand.
func GeneratorForLang(lang string) string {
	switch strings.ToLower(lang) {
	case "rust":
		return "rust"
	case "go":
		return "go"
	case "cpp", "cc":
		return "cpp"
	case "c":
		return "c"
	case "csharp":
		return "csharp"
	default:
		// swift, kotlin, ts, python use canonical C or language generator
		return "c"
	}
}

// GenerateCompanions generates language-specific helper and type marker files.
func GenerateCompanions(opts Options, witPath string) error {
	outDir := opts.Out
	baseName := DeriveBaseName(opts, witPath)
	pascalName := ToPascalCase(baseName)

	pkg := opts.Package
	if pkg == "" && witPath != "" {
		pkg, _ = DiscoverPackage(witPath)
	}

	switch strings.ToLower(opts.Lang) {
	case "swift":
		filename := opts.CompanionFilename
		if filename == "" {
			if pascalName != "" {
				filename = pascalName + ".swift"
			} else {
				filename = "WitBridging.swift"
			}
		} else if !strings.HasSuffix(filename, ".swift") {
			filename += ".swift"
		}

		swiftContent := "// Auto-generated Swift bridging header and protocol markers for WIT\nimport Foundation\n"
		if err := os.WriteFile(filepath.Join(outDir, filename), []byte(swiftContent), 0644); err != nil {
			return err
		}

		modName := opts.ModuleName
		if modName == "" {
			if pascalName != "" {
				modName = pascalName
			} else {
				modName = strings.TrimSuffix(filename, ".swift")
			}
		}

		hName := "structures.h"
		if entries, err := os.ReadDir(outDir); err == nil {
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".h") {
					hName = e.Name()
					break
				}
			}
		}

		moduleMap := fmt.Sprintf("module %s {\n    header \"%s\"\n    export *\n}\n", modName, hName)
		return os.WriteFile(filepath.Join(outDir, "module.modulemap"), []byte(moduleMap), 0644)

	case "kotlin":
		filename := opts.CompanionFilename
		if filename == "" {
			if pascalName != "" {
				filename = pascalName + ".kt"
			} else {
				filename = "WitBindings.kt"
			}
		} else if !strings.HasSuffix(filename, ".kt") {
			filename += ".kt"
		}

		ktPackage := "wit.bindings"
		if pkg != "" {
			ktPackage = strings.ReplaceAll(pkg, ":", ".")
		}
		content := fmt.Sprintf("// Auto-generated Kotlin WASI binding markers for WIT\npackage %s\n", ktPackage)
		return os.WriteFile(filepath.Join(outDir, filename), []byte(content), 0644)

	case "ts":
		filename := opts.CompanionFilename
		if filename == "" {
			filename = "index.d.ts"
		}
		content := "// Auto-generated TypeScript definitions for WIT component\nexport interface WitComponent {\n  readonly [key: string]: unknown;\n}\n"
		if err := os.WriteFile(filepath.Join(outDir, filename), []byte(content), 0644); err != nil {
			return err
		}
		if filename != "index.d.ts" {
			_ = os.WriteFile(filepath.Join(outDir, "index.d.ts"), []byte(content), 0644)
		}
		return nil

	case "python", "py":
		initPy := "# Auto-generated Python bindings for WIT component\n__all__ = []\n"
		initPyi := "# Type stubs for WIT component\nfrom typing import Any\n"
		if err := os.WriteFile(filepath.Join(outDir, "__init__.py"), []byte(initPy), 0644); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(outDir, "__init__.pyi"), []byte(initPyi), 0644)
	}
	return nil
}

// Run executes the WIT bindings generation.
func Run(opts Options) error {
	if err := os.MkdirAll(opts.Out, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", opts.Out, err)
	}

	if len(opts.Srcs) == 0 {
		return fmt.Errorf("no WIT sources specified")
	}

	witPath := opts.Srcs[0]
	if len(opts.Srcs) > 1 {
		// Consolidate into a temporary directory if multiple file sources given
		tmpDir, err := os.MkdirTemp("", "wit-srcs-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		for _, src := range opts.Srcs {
			fi, err := os.Stat(src)
			if err != nil {
				return err
			}
			if fi.IsDir() {
				entries, err := os.ReadDir(src)
				if err != nil {
					return err
				}
				for _, e := range entries {
					if !e.IsDir() && strings.HasSuffix(e.Name(), ".wit") {
						data, err := os.ReadFile(filepath.Join(src, e.Name()))
						if err != nil {
							return err
						}
						if err := os.WriteFile(filepath.Join(tmpDir, e.Name()), data, 0644); err != nil {
							return err
						}
					}
				}
			} else {
				data, err := os.ReadFile(src)
				if err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(tmpDir, filepath.Base(src)), data, 0644); err != nil {
					return err
				}
			}
		}
		witPath = tmpDir
	}

	subcmd := GeneratorForLang(opts.Lang)
	bindgen := opts.Bindgen
	if bindgen == "" {
		bindgen = "wit-bindgen"
	}

	worlds := opts.Worlds
	if len(worlds) == 0 {
		discovered, err := DiscoverWorlds(witPath)
		if err == nil && len(discovered) > 1 {
			worlds = discovered
		}
	}

	var langSpecificFlags []string
	if strings.ToLower(opts.Lang) == "go" {
		goPkg := opts.Package
		if goPkg == "" {
			goPkg, _ = DiscoverPackage(witPath)
		}
		if goPkg != "" {
			if idx := strings.LastIndex(goPkg, ":"); idx != -1 {
				goPkg = goPkg[idx+1:]
			}
			langSpecificFlags = append(langSpecificFlags, "--pkg-name", goPkg)
		}
	} else if strings.ToLower(opts.Lang) == "cpp" || strings.ToLower(opts.Lang) == "cc" {
		cppPrefix := opts.Package
		if cppPrefix == "" {
			cppPrefix, _ = DiscoverPackage(witPath)
		}
		if cppPrefix != "" {
			cppPrefix = strings.ReplaceAll(cppPrefix, ":", "_")
			langSpecificFlags = append(langSpecificFlags, "--internal-prefix", cppPrefix)
		}
	}

	if len(worlds) == 0 {
		args := []string{subcmd, "--out-dir", opts.Out}
		args = append(args, langSpecificFlags...)
		args = append(args, opts.Flags...)
		args = append(args, witPath)

		cmd := exec.Command(bindgen, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// Try fallback to 'c' generator if custom generator was used
			if subcmd != "c" {
				fallbackArgs := []string{"c", "--out-dir", opts.Out}
				fallbackArgs = append(fallbackArgs, opts.Flags...)
				fallbackArgs = append(fallbackArgs, witPath)
				fallbackCmd := exec.Command(bindgen, fallbackArgs...)
				fallbackCmd.Stdout = os.Stdout
				fallbackCmd.Stderr = os.Stderr
				if err2 := fallbackCmd.Run(); err2 != nil {
					return fmt.Errorf("wit-bindgen failed: %w (fallback error: %v)", err, err2)
				}
			} else {
				return fmt.Errorf("wit-bindgen %s failed: %w", subcmd, err)
			}
		}
	} else {
		for _, w := range worlds {
			args := []string{subcmd, "--out-dir", opts.Out, "--world", w}
			args = append(args, langSpecificFlags...)
			args = append(args, opts.Flags...)
			args = append(args, witPath)

			cmd := exec.Command(bindgen, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				if subcmd != "c" {
					fallbackArgs := []string{"c", "--out-dir", opts.Out, "--world", w}
					fallbackArgs = append(fallbackArgs, opts.Flags...)
					fallbackArgs = append(fallbackArgs, witPath)
					fallbackCmd := exec.Command(bindgen, fallbackArgs...)
					fallbackCmd.Stdout = os.Stdout
					fallbackCmd.Stderr = os.Stderr
					if err2 := fallbackCmd.Run(); err2 != nil {
						return fmt.Errorf("wit-bindgen for world %s failed: %w", w, err)
					}
				} else {
					return fmt.Errorf("wit-bindgen %s for world %s failed: %w", subcmd, w, err)
				}
			}
		}
	}

	return GenerateCompanions(opts, witPath)
}
