// Package python provides the self-contained Tree-sitter LanguageConfig for Python.
package python

import (
	"strings"

	"github.com/odvcencio/gotreesitter/grammars/python"
	"tools/common/treesitter/rules"
)

// Config returns the self-contained LanguageConfig for Python.
func Config() *rules.LanguageConfig {
	return &rules.LanguageConfig{
		ID:         rules.LangPython,
		ReportName: string(rules.LangPython),
		Aliases:    []string{"py", "python"},
		Extensions: []string{".py"},
		IsTestFile: func(baseLower string) bool {
			return (strings.HasPrefix(baseLower, "test_") && strings.HasSuffix(baseLower, ".py")) ||
				strings.HasSuffix(baseLower, "_test.py")
		},
		Grammar: python.Language,
		Function: rules.FunctionRules{
			NodeTypes: []string{"function_definition"},
			NameField: "name",
			BodyField: "body",
			BodyTypes: []string{"block"},
		},
		Test: rules.TestRules{
			FuncNamePrefs: []string{"test_"},
		},
		Replacement: rules.ReplacementRules{
			BinaryNodes:   []string{"binary_operator", "boolean_operator", "comparison_operator"},
			UnaryNodes:    []string{"not_operator", "unary_operator"},
			AssignNodes:   []string{"augmented_assignment"},
			BoolTrueNode:  "true",
			BoolFalseNode: "false",
			StringNodes:   []string{"string"},
			BinaryOps: map[string]rules.OpReplacement{
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
				"and": {rules.MutatorLogical, []string{"or"}},
				"or":  {rules.MutatorLogical, []string{"and"}},
			},
			UnaryOps: []string{"not"},
			AssignOps: map[string]string{
				"+=": "-=",
				"-=": "+=",
				"*=": "/=",
				"/=": "*=",
			},
			BoolLiterals: map[string]string{
				"True":  "False",
				"False": "True",
			},
		},
	}
}
