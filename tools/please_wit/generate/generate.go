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
	Bindgen string
	Lang    string
	Out     string
	Srcs    []string
	Worlds  []string
	Flags   []string
}

var worldRegex = regexp.MustCompile(`^\s*world\s+([a-zA-Z0-9_-]+)`)

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
func GenerateCompanions(lang, outDir string) error {
	switch strings.ToLower(lang) {
	case "swift":
		content := "// Auto-generated Swift bridging header and protocol markers for WIT\nimport Foundation\n"
		return os.WriteFile(filepath.Join(outDir, "WitBridging.swift"), []byte(content), 0644)
	case "kotlin":
		content := "// Auto-generated Kotlin WASI binding markers for WIT\npackage wit.bindings\n"
		return os.WriteFile(filepath.Join(outDir, "WitBindings.kt"), []byte(content), 0644)
	case "ts":
		content := "// Auto-generated TypeScript definitions for WIT component\nexport interface WitComponent {\n  readonly [key: string]: unknown;\n}\n"
		return os.WriteFile(filepath.Join(outDir, "index.d.ts"), []byte(content), 0644)
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

	if len(worlds) == 0 {
		args := []string{subcmd, "--out-dir", opts.Out}
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

	return GenerateCompanions(opts.Lang, opts.Out)
}
