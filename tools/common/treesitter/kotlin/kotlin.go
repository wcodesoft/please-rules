// Package kotlin provides the self-contained Tree-sitter LanguageConfig for Kotlin.
package kotlin

import (
	"strings"

	"github.com/odvcencio/gotreesitter/grammars/kotlin"
	"tools/common/treesitter/rules"
)

// Config returns the self-contained LanguageConfig for Kotlin.
func Config() *rules.LanguageConfig {
	return &rules.LanguageConfig{
		ID:         rules.LangKotlin,
		ReportName: string(rules.LangKotlin),
		Aliases:    []string{"kt", "kts", "kotlin"},
		Extensions: []string{".kt", ".kts"},
		IsTestFile: func(baseLower string) bool {
			return strings.HasSuffix(baseLower, "test.kt") || strings.HasSuffix(baseLower, "tests.kt")
		},
		Grammar: kotlin.Language,
		Function: rules.FunctionRules{
			NodeTypes: []string{"function_declaration", "anonymous_function"},
			NameField: "name",
			BodyField: "body",
			BodyTypes: []string{"function_body"},
		},
		Test: rules.TestRules{
			AttrMarkers: []string{"@Test", "@ParameterizedTest"},
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
			},
			UnaryNodes:    []string{"prefix_expression", "unary_expression"},
			UpdateNodes:   []string{"postfix_expression"},
			AssignNodes:   []string{"assignment"},
			BoolTrueNode:  "true",
			BoolFalseNode: "false",
			StringNodes:   []string{"string_literal", "line_string_literal", "multi_line_string_literal"},
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
