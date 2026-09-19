package generate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"tools/please_wit/ast"
)

type Options struct {
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

// ToCamelCase converts identifiers like "two-sum" or "two_sum" to "twoSum".
func ToCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ':' || r == '.'
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

// ToSnakeCase converts identifiers to snake_case.
func ToSnakeCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ':' || r == '.'
	})
	var lowerParts []string
	for _, p := range parts {
		if p != "" {
			lowerParts = append(lowerParts, strings.ToLower(p))
		}
	}
	return strings.Join(lowerParts, "_")
}

// GenerateFromAST parses WIT source directly into an AST and generates native interfaces.
func GenerateFromAST(opts Options, witPath string) error {
	pkg, err := ast.ParsePath(witPath)
	if err != nil {
		return fmt.Errorf("failed to parse WIT definitions from %s: %w", witPath, err)
	}

	if opts.Package != "" {
		if pkg.Namespace == "" && pkg.Name == "" {
			parts := strings.Split(opts.Package, ":")
			if len(parts) == 2 {
				pkg.Namespace = parts[0]
				pkg.Name = parts[1]
			} else {
				pkg.Name = opts.Package
			}
		}
	}

	gen, err := GetGenerator(opts.Lang)
	if err != nil {
		return err
	}

	baseName := DeriveBaseName(opts, witPath)
	files, err := gen.Generate(pkg, opts, baseName)
	if err != nil {
		return err
	}

	outDir := opts.Out
	for _, file := range files {
		targetPath := filepath.Join(outDir, file.Name)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", file.Name, err)
		}
		if err := os.WriteFile(targetPath, []byte(file.Content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", file.Name, err)
		}
	}

	return nil
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

// Run executes the WIT bindings generation using the internal AST parser and generators.
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

	return GenerateFromAST(opts, witPath)
}
