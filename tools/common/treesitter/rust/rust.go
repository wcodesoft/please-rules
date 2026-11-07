// Package rust provides the self-contained Tree-sitter LanguageConfig for Rust.
package rust

import (
	"strings"

	"github.com/odvcencio/gotreesitter/grammars/rust"
	"tools/common/treesitter/rules"
)

// Config returns the self-contained LanguageConfig for Rust.
func Config() *rules.LanguageConfig {
	return &rules.LanguageConfig{
		ID:         rules.LangRust,
		ReportName: string(rules.LangRust),
		Aliases:    []string{"rs", "rust"},
		Extensions: []string{".rs"},
		IsTestFile: func(baseLower string) bool {
			return strings.HasSuffix(baseLower, "_test.rs")
		},
		Grammar: rust.Language,
		Function: rules.FunctionRules{
			NodeTypes: []string{"function_item", "closure_expression"},
			NameField: "name",
			BodyField: "body",
			BodyTypes: []string{"block"},
		},
		Test: rules.TestRules{
			AttrMarkers: []string{"#[test]", "#[tokio::test]", "#[rstest]", "#[cfg(test)]"},
		},
		Replacement: rules.ReplacementRules{
			BinaryNodes:   []string{"binary_expression"},
			UnaryNodes:    []string{"unary_expression"},
			AssignNodes:   []string{"compound_assignment_expr"},
			BoolTrueNode:  "true",
			BoolFalseNode: "false",
			StringNodes:   []string{"string_literal", "raw_string_literal"},
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
