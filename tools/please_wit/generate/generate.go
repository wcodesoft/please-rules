package generate

import (
	"bufio"
	"fmt"
	"io"
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

// resolveWitFiles returns a slice of .wit file paths from a directory or single file path.
func resolveWitFiles(witPath string) ([]string, error) {
	fi, err := os.Stat(witPath)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return []string{witPath}, nil
	}

	entries, err := os.ReadDir(witPath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".wit") {
			files = append(files, filepath.Join(witPath, e.Name()))
		}
	}
	return files, nil
}

// DiscoverWorlds scans all .wit files in a directory or file list and returns declared world names.
func DiscoverWorlds(witPath string) ([]string, error) {
	files, err := resolveWitFiles(witPath)
	if err != nil {
		return nil, err
	}

	var worlds []string
	seen := make(map[string]bool)
	for _, file := range files {
		extracted := parseWorldsFromFile(file, seen)
		worlds = append(worlds, extracted...)
	}

	return worlds, nil
}

func parseWorldsFromFile(file string, seen map[string]bool) []string {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()
	return parseWorldsFromReader(f, seen)
}

func parseWorldsFromReader(r io.Reader, seen map[string]bool) []string {
	var worlds []string
	scanner := bufio.NewScanner(r)
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
	return worlds
}

// DiscoverPackage scans all .wit files in a directory or file list and returns the declared package name if found.
func DiscoverPackage(witPath string) (string, error) {
	files, err := resolveWitFiles(witPath)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if pkg := parsePackageFromFile(file); pkg != "" {
			return pkg, nil
		}
	}

	return "", nil
}

func parsePackageFromFile(file string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	return parsePackageFromContent(data)
}

func parsePackageFromContent(data []byte) string {
	matches := packageRegex.FindSubmatch(data)
	if len(matches) > 1 {
		return string(matches[1])
	}
	return ""
}

// splitIdentifier splits identifiers by common delimiters (_, -, :, .).
func splitIdentifier(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ':' || r == '.'
	})
}

// ToPascalCase converts names like "structures" or "two_sum" or "two-sum" to "Structures" or "TwoSum".
func ToPascalCase(s string) string {
	parts := splitIdentifier(s)
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// ToCamelCase converts identifiers like "two-sum" or "two_sum" to "twoSum".
func ToCamelCase(s string) string {
	parts := splitIdentifier(s)
	if len(parts) == 0 {
		return s
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(parts[0]))
	for _, p := range parts[1:] {
		if len(p) > 0 {
			b.WriteString(strings.ToUpper(p[:1]))
			b.WriteString(strings.ToLower(p[1:]))
		}
	}
	return b.String()
}

// ToSnakeCase converts identifiers to snake_case.
func ToSnakeCase(s string) string {
	parts := splitIdentifier(s)
	var lowerParts []string
	for _, p := range parts {
		if p != "" {
			lowerParts = append(lowerParts, strings.ToLower(p))
		}
	}
	return strings.Join(lowerParts, "_")
}

// DeriveBaseName determines the logical base name for outputs based on companion filename, package, worlds, or directory name.
func DeriveBaseName(opts Options, witPath string) string {
	if name := deriveFromCompanion(opts.CompanionFilename); name != "" {
		return name
	}
	if name := deriveFromPackage(opts.Package, witPath); name != "" {
		return name
	}
	if name := deriveFromWitPath(witPath); name != "" {
		return name
	}
	return "Wit"
}

func deriveFromCompanion(companionFilename string) string {
	if companionFilename == "" {
		return ""
	}
	base := filepath.Base(companionFilename)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func deriveFromPackage(pkg string, witPath string) string {
	if pkg == "" && witPath != "" {
		pkg, _ = DiscoverPackage(witPath)
	}
	if pkg == "" {
		return ""
	}
	if idx := strings.LastIndex(pkg, ":"); idx != -1 {
		return pkg[idx+1:]
	}
	return pkg
}

func deriveFromWitPath(witPath string) string {
	if witPath == "" {
		return ""
	}
	worlds, _ := DiscoverWorlds(witPath)
	if len(worlds) > 0 {
		w := worlds[0]
		w = strings.TrimSuffix(w, "-world")
		w = strings.TrimSuffix(w, "_world")
		return w
	}

	base := filepath.Base(witPath)
	return strings.TrimSuffix(base, "_wit")
}

// GenerateFromAST parses WIT source directly into an AST and generates native interfaces.
func GenerateFromAST(opts Options, witPath string) error {
	pkg, err := ast.ParsePath(witPath)
	if err != nil {
		return fmt.Errorf("failed to parse WIT definitions from %s: %w", witPath, err)
	}

	applyPackageOverride(pkg, opts.Package)

	gen, err := GetGenerator(opts.Lang)
	if err != nil {
		return err
	}

	baseName := DeriveBaseName(opts, witPath)
	files, err := gen.Generate(pkg, opts, baseName)
	if err != nil {
		return err
	}

	return writeOutputFiles(opts.Out, files)
}

func applyPackageOverride(pkg *ast.Package, pkgOverride string) {
	if pkgOverride == "" {
		return
	}
	if pkg.Namespace != "" || pkg.Name != "" {
		return
	}
	parts := strings.Split(pkgOverride, ":")
	if len(parts) == 2 {
		pkg.Namespace = parts[0]
		pkg.Name = parts[1]
	} else {
		pkg.Name = pkgOverride
	}
}

func writeOutputFiles(outDir string, files []OutputFile) error {
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

// Run executes the WIT bindings generation using the internal AST parser and generators.
func Run(opts Options) error {
	if err := os.MkdirAll(opts.Out, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", opts.Out, err)
	}

	if len(opts.Srcs) == 0 {
		return fmt.Errorf("no WIT sources specified")
	}

	witPath, cleanup, err := prepareWitSource(opts.Srcs)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return GenerateFromAST(opts, witPath)
}

func prepareWitSource(srcs []string) (string, func(), error) {
	if len(srcs) == 1 {
		return srcs[0], nil, nil
	}

	tmpDir, err := os.MkdirTemp("", "wit-srcs-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	if err := copySourcesToDir(srcs, tmpDir); err != nil {
		cleanup()
		return "", nil, err
	}
	return tmpDir, cleanup, nil
}

func copySourcesToDir(srcs []string, destDir string) error {
	for _, src := range srcs {
		if err := copySourceEntry(src, destDir); err != nil {
			return err
		}
	}
	return nil
}

func copySourceEntry(src string, destDir string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return copyWitDirEntries(src, destDir)
	}
	return copyFile(src, filepath.Join(destDir, filepath.Base(src)))
}

func copyWitDirEntries(srcDir string, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".wit") {
			srcFile := filepath.Join(srcDir, e.Name())
			destFile := filepath.Join(destDir, e.Name())
			if err := copyFile(srcFile, destFile); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(srcPath, destPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	return os.WriteFile(destPath, data, 0644)
}
