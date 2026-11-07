// Package treesitter provides pure-Go multi-language CST parsing, function/test
// discovery, and syntax-directed mutation site extraction built on gotreesitter.
//
// Each supported language is defined in its own dedicated subpackage under
// tools/common/treesitter/<lang> (go, rust, typescript, kotlin, swift, python).
package treesitter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/odvcencio/gotreesitter"
	golang "tools/common/treesitter/go"
	"tools/common/treesitter/kotlin"
	"tools/common/treesitter/python"
	"tools/common/treesitter/rules"
	"tools/common/treesitter/rust"
	"tools/common/treesitter/swift"
	"tools/common/treesitter/typescript"
)

// Re-export core types and constants from rules so callers only need to import
// "tools/common/treesitter".
type (
	LangID           = rules.LangID
	MutatorCategory  = rules.MutatorCategory
	OpReplacement    = rules.OpReplacement
	FunctionRules    = rules.FunctionRules
	TestRules        = rules.TestRules
	ReplacementRules = rules.ReplacementRules
	LanguageConfig   = rules.LanguageConfig
	Site             = rules.Site
)

const (
	LangGo         = rules.LangGo
	LangRust       = rules.LangRust
	LangTypeScript = rules.LangTypeScript
	LangTSX        = rules.LangTSX
	LangJavaScript = rules.LangJavaScript
	LangKotlin     = rules.LangKotlin
	LangSwift      = rules.LangSwift
	LangPython     = rules.LangPython

	MutatorArithmetic     = rules.MutatorArithmetic
	MutatorComparison     = rules.MutatorComparison
	MutatorEquality       = rules.MutatorEquality
	MutatorLogical        = rules.MutatorLogical
	MutatorBooleanLiteral = rules.MutatorBooleanLiteral
	MutatorUnary          = rules.MutatorUnary
	MutatorAssignment     = rules.MutatorAssignment
)

// AllMutatorCategories lists all default mutator categories in deterministic order.
var AllMutatorCategories = rules.AllMutatorCategories

// Apply returns a new byte slice with the site's [StartByte:EndByte] replaced by Replacement.
func Apply(src []byte, s Site) []byte {
	return rules.Apply(src, s)
}

var (
	allLanguages = []*LanguageConfig{
		golang.Config(),
		rust.Config(),
		typescript.Config(),
		typescript.TSXConfig(),
		typescript.JavaScriptConfig(),
		kotlin.Config(),
		swift.Config(),
		python.Config(),
	}

	configsByID    = indexByID(allLanguages)
	configsByAlias = indexByAlias(allLanguages)
	configsByExt   = indexByExt(allLanguages)
)

func indexByID(cfgs []*LanguageConfig) map[LangID]*LanguageConfig {
	m := make(map[LangID]*LanguageConfig, len(cfgs))
	for _, cfg := range cfgs {
		m[cfg.ID] = cfg
	}
	return m
}

func indexByAlias(cfgs []*LanguageConfig) map[string]*LanguageConfig {
	m := make(map[string]*LanguageConfig, len(cfgs)*2)
	for _, cfg := range cfgs {
		for _, alias := range cfg.Aliases {
			m[alias] = cfg
		}
	}
	return m
}

func indexByExt(cfgs []*LanguageConfig) map[string]*LanguageConfig {
	m := make(map[string]*LanguageConfig, len(cfgs)*2)
	for _, cfg := range cfgs {
		for _, ext := range cfg.Extensions {
			m[ext] = cfg
		}
	}
	return m
}

// ResolveLanguage resolves the self-contained LanguageConfig from an explicit
// language name or, when explicit is empty, from the file extension of path.
func ResolveLanguage(path, explicit string) (*LanguageConfig, error) {
	trimmed := strings.ToLower(strings.TrimSpace(explicit))
	if trimmed != "" {
		return resolveByAlias(path, trimmed, explicit)
	}
	return resolveByExtension(path)
}

func resolveByAlias(path, normalizedAlias, rawExplicit string) (*LanguageConfig, error) {
	cfg, ok := configsByAlias[normalizedAlias]
	if !ok {
		return nil, fmt.Errorf("unsupported language %q", rawExplicit)
	}
	if cfg.ID == LangTypeScript && strings.HasSuffix(strings.ToLower(path), ".tsx") {
		return configsByID[LangTSX], nil
	}
	return cfg, nil
}

func resolveByExtension(path string) (*LanguageConfig, error) {
	ext := strings.ToLower(filepath.Ext(path))
	cfg, ok := configsByExt[ext]
	if !ok {
		return nil, fmt.Errorf("cannot detect language for %q (unsupported extension %q)", path, ext)
	}
	return cfg, nil
}

// LookupLanguage returns the registered LanguageConfig for id.
func LookupLanguage(id LangID) (*LanguageConfig, error) {
	cfg, ok := configsByID[id]
	if !ok {
		return nil, fmt.Errorf("unknown language %q", id)
	}
	return cfg, nil
}

// DetectLanguage infers the LangID from a file path extension or explicit language name.
func DetectLanguage(path, explicit string) (LangID, error) {
	cfg, err := ResolveLanguage(path, explicit)
	if err != nil {
		return "", err
	}
	return cfg.ID, nil
}

// IsTestFile reports whether a filename follows conventional test file naming
// for any supported language and should be skipped when scanning source directories.
func IsTestFile(path string) bool {
	baseLower := strings.ToLower(filepath.Base(path))
	for _, cfg := range allLanguages {
		if cfg.IsTestFile != nil && cfg.IsTestFile(baseLower) {
			return true
		}
	}
	return false
}

// Parse parses src using the grammar for id and returns the CST and Language.
func Parse(id LangID, src []byte) (*gotreesitter.Tree, *gotreesitter.Language, error) {
	cfg, err := LookupLanguage(id)
	if err != nil {
		return nil, nil, err
	}
	return cfg.Parse(src)
}

// Discover parses src for language id and returns all candidate mutation sites
// located inside non-test executable function bodies, filtered by allowedMutators
// (if non-empty).
func Discover(id LangID, file string, src []byte, allowedMutators map[MutatorCategory]bool) ([]Site, error) {
	cfg, err := LookupLanguage(id)
	if err != nil {
		return nil, err
	}
	return cfg.Discover(file, src, allowedMutators)
}
