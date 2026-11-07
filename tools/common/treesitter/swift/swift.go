// Package swift provides the self-contained Tree-sitter LanguageConfig for Swift.
package swift

import (
	"strings"

	"github.com/odvcencio/gotreesitter/grammars/swift"
	"tools/common/treesitter/rules"
)

// Config returns the self-contained LanguageConfig for Swift.
func Config() *rules.LanguageConfig {
	return &rules.LanguageConfig{
		ID:         rules.LangSwift,
		ReportName: string(rules.LangSwift),
		Aliases:    []string{"swift"},
		Extensions: []string{".swift"},
		IsTestFile: func(baseLower string) bool {
			return strings.HasSuffix(baseLower, "test.swift") || strings.HasSuffix(baseLower, "tests.swift")
		},
		Grammar: swift.Language,
		Function: rules.FunctionRules{
			NodeTypes:   []string{"function_declaration", "init_declaration"},
			NameField:   "name",
			BodyField:   "body",
			BodyTypes:   []string{"function_body"},
			SkipParents: []string{"protocol_body"},
		},
		Test: rules.TestRules{
			AttrMarkers:   []string{"@Test"},
			FuncNamePrefs: []string{"test"},
		},
		Replacement: rules.ReplacementRules{
			BinaryNodes: []string{
				"additive_expression",
				"multiplicative_expression",
				"comparison_expression",
				"equality_expression",
				"conjunction_expression",
				"disjunction_expression",
				"binary_expression",
				"infix_expression",
			},
			UnaryNodes:    []string{"prefix_expression"},
			UpdateNodes:   []string{"postfix_expression"},
			AssignNodes:   []string{"assignment", "directly_assignable_expression"},
			BoolTrueNode:  "true",
			BoolFalseNode: "false",
			StringNodes:   []string{"line_string_literal", "multi_line_string_literal"},
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
