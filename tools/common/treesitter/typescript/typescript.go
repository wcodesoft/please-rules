// Package typescript provides self-contained Tree-sitter LanguageConfigs for
// TypeScript, TSX, and JavaScript.
package typescript

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars/javascript"
	"github.com/odvcencio/gotreesitter/grammars/tsx"
	tsgrammar "github.com/odvcencio/gotreesitter/grammars/typescript"
	"tools/common/treesitter/rules"
)

var testSuffixes = []string{
	".test.ts", ".spec.ts", "_test.ts",
	".test.tsx", ".spec.tsx", "_test.tsx",
	".test.js", ".spec.js", "_test.js",
	".test.jsx", ".spec.jsx", "_test.jsx",
}

func isTestFile(baseLower string) bool {
	for _, suffix := range testSuffixes {
		if strings.HasSuffix(baseLower, suffix) {
			return true
		}
	}
	return false
}

func binaryOps() map[string]rules.OpReplacement {
	return map[string]rules.OpReplacement{
		"+":   {rules.MutatorArithmetic, []string{"-"}},
		"-":   {rules.MutatorArithmetic, []string{"+"}},
		"*":   {rules.MutatorArithmetic, []string{"/"}},
		"/":   {rules.MutatorArithmetic, []string{"*"}},
		"%":   {rules.MutatorArithmetic, []string{"*"}},
		"<":   {rules.MutatorComparison, []string{"<=", ">="}},
		"<=":  {rules.MutatorComparison, []string{"<", ">"}},
		">":   {rules.MutatorComparison, []string{">=", "<="}},
		">=":  {rules.MutatorComparison, []string{">", "<"}},
		"==":  {rules.MutatorEquality, []string{"!="}},
		"!=":  {rules.MutatorEquality, []string{"=="}},
		"===": {rules.MutatorEquality, []string{"!=="}},
		"!==": {rules.MutatorEquality, []string{"==="}},
		"&&":  {rules.MutatorLogical, []string{"||"}},
		"||":  {rules.MutatorLogical, []string{"&&"}},
	}
}

func newFamilyConfig(id rules.LangID, aliases, exts []string, grammar func() *gotreesitter.Language) *rules.LanguageConfig {
	return &rules.LanguageConfig{
		ID:         id,
		ReportName: string(rules.LangTypeScript),
		Aliases:    aliases,
		Extensions: exts,
		IsTestFile: isTestFile,
		Grammar:    grammar,
		Function: rules.FunctionRules{
			NodeTypes: []string{"function_declaration", "method_definition", "arrow_function", "function_expression"},
			NameField: "name",
			BodyField: "body",
			BodyTypes: []string{"statement_block"},
		},
		Test: rules.TestRules{
			CallNames: []string{"describe", "it", "test", "Deno.test"},
		},
		Replacement: rules.ReplacementRules{
			BinaryNodes:   []string{"binary_expression"},
			UnaryNodes:    []string{"unary_expression"},
			UpdateNodes:   []string{"update_expression"},
			AssignNodes:   []string{"augmented_assignment_expression"},
			BoolTrueNode:  "true",
			BoolFalseNode: "false",
			StringNodes:   []string{"string", "template_string"},
			BinaryOps:     binaryOps(),
			UnaryOps:      []string{"!"},
			UpdateOps: map[string]string{
				"++": "--",
				"--": "++",
			},
			AssignOps: map[string]string{
				"+=": "-=",
				"-=": "+=",
				"*=": "/=",
				"/=": "*=",
			},
			BoolLiterals: map[string]string{
				"true":  "false",
				"false": "true",
			},
		},
	}
}

// Config returns the self-contained LanguageConfig for TypeScript (.ts).
func Config() *rules.LanguageConfig {
	return newFamilyConfig(
		rules.LangTypeScript,
		[]string{"ts", "typescript"},
		[]string{".ts", ".mts", ".cts"},
		tsgrammar.Language,
	)
}

// TSXConfig returns the self-contained LanguageConfig for TSX (.tsx).
func TSXConfig() *rules.LanguageConfig {
	return newFamilyConfig(
		rules.LangTSX,
		[]string{"tsx"},
		[]string{".tsx"},
		tsx.Language,
	)
}

// JavaScriptConfig returns the self-contained LanguageConfig for JavaScript (.js, .jsx).
func JavaScriptConfig() *rules.LanguageConfig {
	return newFamilyConfig(
		rules.LangJavaScript,
		[]string{"js", "jsx", "javascript"},
		[]string{".js", ".jsx", ".mjs", ".cjs"},
		javascript.Language,
	)
}
