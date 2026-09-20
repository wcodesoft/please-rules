package generate

import (
	"fmt"
	"strings"
	"tools/please_wit/ast"
)

// OutputFile represents a single generated output file.
type OutputFile struct {
	Name    string // Relative filename within the output directory (e.g. "Structures.kt", "module.modulemap")
	Content string
}

// Generator represents a language-specific AST code generator.
type Generator interface {
	Name() string
	MapWitType(t *ast.TypeRef) string
	Generate(pkg *ast.Package, opts Options, baseName string) ([]OutputFile, error)
}

var registry = map[string]Generator{
	"kotlin":     &KotlinGenerator{},
	"kt":         &KotlinGenerator{},
	"swift":      &SwiftGenerator{},
	"ts":         &TypeScriptGenerator{},
	"typescript": &TypeScriptGenerator{},
	"python":     &PythonGenerator{},
	"py":         &PythonGenerator{},
	"rust":       &RustGenerator{},
	"rs":         &RustGenerator{},
	"go":         &GoGenerator{},
	"golang":     &GoGenerator{},
	"cpp":        &CppGenerator{},
	"cc":         &CppGenerator{},
	"c":          &CppGenerator{},
	"cxx":        &CppGenerator{},
}

// GetGenerator retrieves a Generator instance for the specified language name or alias.
func GetGenerator(lang string) (Generator, error) {
	if gen, ok := registry[strings.ToLower(lang)]; ok {
		return gen, nil
	}
	return nil, fmt.Errorf("unsupported language: %s", lang)
}
