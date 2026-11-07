// Package rules defines the core language configuration structs, CST rule specs,
// and syntax-directed mutation discovery engine shared by all language subpackages.
package rules

import (
	"fmt"

	"github.com/odvcencio/gotreesitter"
)

// LangID identifies a supported source language.
type LangID string

const (
	LangGo         LangID = "go"
	LangRust       LangID = "rust"
	LangTypeScript LangID = "ts"
	LangTSX        LangID = "tsx"
	LangJavaScript LangID = "js"
	LangKotlin     LangID = "kotlin"
	LangSwift      LangID = "swift"
	LangPython     LangID = "python"
)

// MutatorCategory names a class of syntax-directed source mutations.
type MutatorCategory string

const (
	MutatorArithmetic     MutatorCategory = "arithmetic"
	MutatorComparison     MutatorCategory = "comparison"
	MutatorEquality       MutatorCategory = "equality"
	MutatorLogical        MutatorCategory = "logical"
	MutatorBooleanLiteral MutatorCategory = "boolean_literal"
	MutatorUnary          MutatorCategory = "unary"
	MutatorAssignment     MutatorCategory = "assignment"
)

// AllMutatorCategories lists all default mutator categories in deterministic order.
var AllMutatorCategories = []MutatorCategory{
	MutatorArithmetic,
	MutatorComparison,
	MutatorEquality,
	MutatorLogical,
	MutatorBooleanLiteral,
	MutatorUnary,
	MutatorAssignment,
}

// OpReplacement maps an operator token to its mutator category and replacement tokens.
type OpReplacement struct {
	Category     MutatorCategory
	Replacements []string
}

// FunctionRules configures how function and method declarations are recognized in a CST.
type FunctionRules struct {
	NodeTypes   []string
	NameField   string
	BodyField   string
	BodyTypes   []string
	SkipParents []string
}

// TestRules configures how in-file test blocks, test functions, and test calls are
// recognized so that test code embedded inside source files is never mutated.
type TestRules struct {
	AttrMarkers   []string
	CallNames     []string
	FuncNamePrefs []string
}

// ReplacementRules configures the CST node types and operator/literal replacement tables
// for a specific language.
type ReplacementRules struct {
	BinaryNodes   []string
	UnaryNodes    []string
	UpdateNodes   []string
	AssignNodes   []string
	BoolTrueNode  string
	BoolFalseNode string
	StringNodes   []string
	BinaryOps     map[string]OpReplacement
	UnaryOps      []string
	UpdateOps     map[string]string
	AssignOps     map[string]string
	BoolLiterals  map[string]string
}

// LanguageConfig encapsulates the Tree-sitter grammar loader, file matching rules,
// function/test structural rules, and mutation replacement rules for a language.
type LanguageConfig struct {
	ID          LangID
	ReportName  string
	Aliases     []string
	Extensions  []string
	IsTestFile  func(baseLower string) bool
	Grammar     func() *gotreesitter.Language
	Function    FunctionRules
	Test        TestRules
	Replacement ReplacementRules
}

// Parse parses src using the language config's grammar and returns the CST and Language.
func (c *LanguageConfig) Parse(src []byte) (*gotreesitter.Tree, *gotreesitter.Language, error) {
	if c == nil || c.Grammar == nil {
		return nil, nil, fmt.Errorf("language config has no grammar")
	}
	lang := c.Grammar()
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, nil, err
	}
	return tree, lang, nil
}
