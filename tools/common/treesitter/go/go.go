// Package golang provides the self-contained Tree-sitter LanguageConfig for Go.
package golang

import (
	"strings"

	golang "github.com/odvcencio/gotreesitter/grammars/go"
	"tools/common/treesitter/rules"
)

// Config returns the self-contained LanguageConfig for Go.
func Config() *rules.LanguageConfig {
	return &rules.LanguageConfig{
		ID:         rules.LangGo,
		ReportName: string(rules.LangGo),
		Aliases:    []string{"go", "golang"},
		Extensions: []string{".go"},
		IsTestFile: func(baseLower string) bool {
			return strings.HasSuffix(baseLower, "_test.go")
		},
		Grammar: golang.Language,
		Function: rules.FunctionRules{
			NodeTypes: []string{"function_declaration", "method_declaration", "func_literal"},
			NameField: "name",
			BodyField: "body",
			BodyTypes: []string{"block"},
		},
		Test: rules.TestRules{
			FuncNamePrefs: []string{"Test", "Benchmark", "Fuzz", "Example"},
			CallNames:     []string{"t.Run", "b.Run"},
		},
		Replacement: rules.ReplacementRules{
			BinaryNodes:   []string{"binary_expression"},
			UnaryNodes:    []string{"unary_expression"},
			UpdateNodes:   []string{"inc_statement", "dec_statement"},
			AssignNodes:   []string{"assignment_statement"},
			BoolTrueNode:  "true",
			BoolFalseNode: "false",
			StringNodes:   []string{"interpreted_string_literal", "raw_string_literal"},
			BinaryOps: map[string]rules.OpReplacement{
				"+":  {rules.MutatorArithmetic, []string{"-"}},
				"-":  {rules.MutatorArithmetic, []string{"+"}},
				"*":  {rules.MutatorArithmetic, []string{"/"}},
				"/":  {rules.MutatorArithmetic, []string{"*"}},
				"%":  {rules.MutatorArithmetic, []string{"*"}},
				"<":  {rules.MutatorComparison, []string{"<=", ">="}},
				"<=": {rules.MutatorComparison, []string{"<", ">"}},
				">":  {rules.MutatorComparison, []string{">=", "<="}},
				">=": {rules.MutatorComparison, []string{">", "<"}},
				"==": {rules.MutatorEquality, []string{"!="}},
				"!=": {rules.MutatorEquality, []string{"=="}},
				"&&": {rules.MutatorLogical, []string{"||"}},
				"||": {rules.MutatorLogical, []string{"&&"}},
			},
			UnaryOps: []string{"!"},
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
