package ast

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLexer_Punctuation(t *testing.T) {
	input := ": ; , { } ( ) < > = @ ->"
	lexer := NewLexer(input)

	expected := []struct {
		tokType TokenType
		val     string
	}{
		{TokenColon, ":"},
		{TokenSemicolon, ";"},
		{TokenComma, ","},
		{TokenLBrace, "{"},
		{TokenRBrace, "}"},
		{TokenLParen, "("},
		{TokenRParen, ")"},
		{TokenLAngle, "<"},
		{TokenRAngle, ">"},
		{TokenEquals, "="},
		{TokenAt, "@"},
		{TokenArrow, "->"},
		{TokenEOF, ""},
	}

	for i, exp := range expected {
		tok := lexer.NextToken()
		if tok.Type != exp.tokType {
			t.Errorf("step %d: expected token type %v, got %v (%q)", i, exp.tokType, tok.Type, tok.Value)
		}
		if tok.Value != exp.val {
			t.Errorf("step %d: expected value %q, got %q", i, exp.val, tok.Value)
		}
	}
}

func TestLexer_IdentifiersAndNumbers(t *testing.T) {
	input := "identifier kebab-case_name _private %interface 123 0.2.1"
	lexer := NewLexer(input)

	expected := []string{
		"identifier",
		"kebab-case_name",
		"_private",
		"%interface",
		"123",
		"0.2.1",
	}

	for _, exp := range expected {
		tok := lexer.NextToken()
		if tok.Type != TokenIdent {
			t.Errorf("expected TokenIdent for %q, got %v", exp, tok.Type)
		}
		if tok.Value != exp {
			t.Errorf("expected value %q, got %q", exp, tok.Value)
		}
	}

	last := lexer.NextToken()
	if last.Type != TokenEOF {
		t.Errorf("expected EOF, got %v", last.Type)
	}
}

func TestLexer_CommentsAndWhitespace(t *testing.T) {
	input := `
	// Single line comment
	pkg1
	// Another comment with symbols: -> = {}
	pkg2
	/* Multi
	   line
	   comment */
	pkg3
	/*** Star comment ***/
	pkg4
	`
	lexer := NewLexer(input)

	expected := []string{"pkg1", "pkg2", "pkg3", "pkg4"}
	for _, exp := range expected {
		tok := lexer.NextToken()
		if tok.Type != TokenIdent || tok.Value != exp {
			t.Errorf("expected ident %q, got %+v", exp, tok)
		}
	}

	if tok := lexer.NextToken(); tok.Type != TokenEOF {
		t.Errorf("expected EOF, got %+v", tok)
	}
}

func TestLexer_HyphenMinus(t *testing.T) {
	input := "- ->"
	lexer := NewLexer(input)

	tok1 := lexer.NextToken()
	if tok1.Type != TokenIdent || tok1.Value != "-" {
		t.Errorf("expected '-' ident, got %+v", tok1)
	}

	tok2 := lexer.NextToken()
	if tok2.Type != TokenArrow || tok2.Value != "->" {
		t.Errorf("expected '->' arrow, got %+v", tok2)
	}
}

func TestParser_PackageSyntax(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedNS  string
		expectedPkg string
		expectedVer string
	}{
		{
			name:        "full package with version",
			input:       "package my-org:my-pkg@1.2.3;",
			expectedNS:  "my-org",
			expectedPkg: "my-pkg",
			expectedVer: "1.2.3",
		},
		{
			name:        "package without version",
			input:       "package org:pkg;",
			expectedNS:  "org",
			expectedPkg: "pkg",
			expectedVer: "",
		},
		{
			name:        "single identifier package",
			input:       "package standalone;",
			expectedNS:  "",
			expectedPkg: "standalone",
			expectedVer: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := ParseContent(tt.input)
			if err != nil {
				t.Fatalf("ParseContent failed: %v", err)
			}
			if pkg.Namespace != tt.expectedNS {
				t.Errorf("expected namespace %q, got %q", tt.expectedNS, pkg.Namespace)
			}
			if pkg.Name != tt.expectedPkg {
				t.Errorf("expected name %q, got %q", tt.expectedPkg, pkg.Name)
			}
			if pkg.Version != tt.expectedVer {
				t.Errorf("expected version %q, got %q", tt.expectedVer, pkg.Version)
			}
		})
	}
}

func TestParser_EmptyInterface(t *testing.T) {
	input := `
	package test:pkg;
	interface empty-api {}
	`
	pkg, err := ParseContent(input)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}
	if len(pkg.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(pkg.Interfaces))
	}
	if pkg.Interfaces[0].Name != "empty-api" {
		t.Errorf("expected empty-api, got %q", pkg.Interfaces[0].Name)
	}
	if len(pkg.Interfaces[0].Functions) != 0 {
		t.Errorf("expected 0 functions, got %d", len(pkg.Interfaces[0].Functions))
	}
}

func TestParser_FunctionVariants(t *testing.T) {
	input := `
	package test:pkg;

	interface api {
		no-args-no-ret: func();
		no-args-with-ret: func() -> string;
		single-arg-no-ret: func(msg: string);
		multiple-args-with-ret: func(a: s32, b: s32, flag: bool) -> result<s32, string>;
	}
	`
	pkg, err := ParseContent(input)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	iface := pkg.Interfaces[0]
	if len(iface.Functions) != 4 {
		t.Fatalf("expected 4 functions, got %d", len(iface.Functions))
	}

	// 1. no-args-no-ret
	fn0 := iface.Functions[0]
	if fn0.Name != "no-args-no-ret" || len(fn0.Params) != 0 || fn0.Results != nil {
		t.Errorf("unexpected fn0: %+v", fn0)
	}

	// 2. no-args-with-ret
	fn1 := iface.Functions[1]
	if fn1.Name != "no-args-with-ret" || len(fn1.Params) != 0 || fn1.Results == nil || fn1.Results.Name != "string" {
		t.Errorf("unexpected fn1: %+v", fn1)
	}

	// 3. single-arg-no-ret
	fn2 := iface.Functions[2]
	if fn2.Name != "single-arg-no-ret" || len(fn2.Params) != 1 || fn2.Params[0].Name != "msg" || fn2.Results != nil {
		t.Errorf("unexpected fn2: %+v", fn2)
	}

	// 4. multiple-args-with-ret
	fn3 := iface.Functions[3]
	if fn3.Name != "multiple-args-with-ret" || len(fn3.Params) != 3 {
		t.Errorf("unexpected fn3: %+v", fn3)
	}
	if fn3.Results == nil || fn3.Results.Kind != KindResult || len(fn3.Results.TypeArgs) != 2 {
		t.Errorf("unexpected fn3 results: %+v", fn3.Results)
	}
}

func TestParser_CompoundAndPrimitiveTypes(t *testing.T) {
	input := `
	package test:types;

	interface type-matrix {
		test-primitives: func(
			p1: u8, p2: u16, p3: u32, p4: u64,
			p5: s8, p6: s16, p7: s32, p8: s64,
			p9: f32, p10: f64, p11: char, p12: bool, p13: string
		);

		test-bare-containers: func(l: list, o: option, r: result, tup: tuple);

		test-tuples: func(t1: tuple<s32, string, bool>);

		test-result-blank: func() -> result<_, string>;
		test-result-single: func() -> result<s32>;

		test-nested: func(matrix: list<list<s32>>, complex: list<option<result<s32, string>>>);
	}
	`
	pkg, err := ParseContent(input)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	iface := pkg.Interfaces[0]
	if len(iface.Functions) != 6 {
		t.Fatalf("expected 6 functions, got %d", len(iface.Functions))
	}

	// Primitives
	fnPrim := iface.Functions[0]
	if len(fnPrim.Params) != 13 {
		t.Fatalf("expected 13 primitive params, got %d", len(fnPrim.Params))
	}
	for _, p := range fnPrim.Params {
		if p.Type.Kind != KindPrimitive {
			t.Errorf("expected primitive kind for %s (%s), got %v", p.Name, p.Type.Name, p.Type.Kind)
		}
	}

	// Bare containers
	fnBare := iface.Functions[1]
	if len(fnBare.Params) != 4 {
		t.Fatalf("expected 4 bare params, got %d", len(fnBare.Params))
	}
	if fnBare.Params[0].Type.Kind != KindList || len(fnBare.Params[0].Type.TypeArgs) != 0 {
		t.Errorf("expected bare list, got %+v", fnBare.Params[0].Type)
	}
	if fnBare.Params[1].Type.Kind != KindOption || len(fnBare.Params[1].Type.TypeArgs) != 0 {
		t.Errorf("expected bare option, got %+v", fnBare.Params[1].Type)
	}
	if fnBare.Params[2].Type.Kind != KindResult || len(fnBare.Params[2].Type.TypeArgs) != 0 {
		t.Errorf("expected bare result, got %+v", fnBare.Params[2].Type)
	}
	if fnBare.Params[3].Type.Kind != KindTuple || len(fnBare.Params[3].Type.TypeArgs) != 0 {
		t.Errorf("expected bare tuple, got %+v", fnBare.Params[3].Type)
	}

	// Tuples
	fnTup := iface.Functions[2]
	tupType := fnTup.Params[0].Type
	if tupType.Kind != KindTuple || len(tupType.TypeArgs) != 3 {
		t.Errorf("expected tuple with 3 args, got %+v", tupType)
	}
	if tupType.String() != "tuple<s32, string, bool>" {
		t.Errorf("expected string representation 'tuple<s32, string, bool>', got %q", tupType.String())
	}

	// result<_, E>
	fnResBlank := iface.Functions[3]
	if fnResBlank.Results == nil || fnResBlank.Results.Kind != KindResult || len(fnResBlank.Results.TypeArgs) != 2 {
		t.Fatalf("expected result<_, string>, got %+v", fnResBlank.Results)
	}
	if fnResBlank.Results.TypeArgs[0].Name != "_" {
		t.Errorf("expected first arg '_', got %q", fnResBlank.Results.TypeArgs[0].Name)
	}

	// result<T> (single argument)
	fnResSingle := iface.Functions[4]
	if fnResSingle.Results == nil || fnResSingle.Results.Kind != KindResult || len(fnResSingle.Results.TypeArgs) != 1 {
		t.Fatalf("expected result<s32>, got %+v", fnResSingle.Results)
	}

	// Nested types
	fnNested := iface.Functions[5]
	matParam := fnNested.Params[0].Type
	if matParam.Kind != KindList || matParam.TypeArgs[0].Kind != KindList {
		t.Errorf("expected nested list<list<s32>>, got %s", matParam.String())
	}
}

func TestParser_RecordsAndEnumsDetailed(t *testing.T) {
	input := `
	package app:models;

	interface entities {
		record empty-rec {}

		record user {
			id: u64,
			username: string,
			roles: list<string>,
		}

		enum empty-enum {}

		enum role {
			admin,
			viewer,
			editor,
		}

		type user-alias = user;
		type id-alias = u64;
	}
	`
	pkg, err := ParseContent(input)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	iface := pkg.Interfaces[0]

	// Records
	if len(iface.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(iface.Records))
	}
	if iface.Records[0].Name != "empty-rec" || len(iface.Records[0].Fields) != 0 {
		t.Errorf("unexpected empty record: %+v", iface.Records[0])
	}
	userRec := iface.Records[1]
	if userRec.Name != "user" || len(userRec.Fields) != 3 {
		t.Fatalf("unexpected user record: %+v", userRec)
	}
	if userRec.Fields[0].Name != "id" || userRec.Fields[0].Type.Name != "u64" {
		t.Errorf("field 0 mismatch: %+v", userRec.Fields[0])
	}
	if userRec.Fields[2].Name != "roles" || userRec.Fields[2].Type.Kind != KindList {
		t.Errorf("field 2 mismatch: %+v", userRec.Fields[2])
	}

	// Enums
	if len(iface.Enums) != 2 {
		t.Fatalf("expected 2 enums, got %d", len(iface.Enums))
	}
	if iface.Enums[0].Name != "empty-enum" || len(iface.Enums[0].Cases) != 0 {
		t.Errorf("unexpected empty enum: %+v", iface.Enums[0])
	}
	roleEnum := iface.Enums[1]
	if roleEnum.Name != "role" || len(roleEnum.Cases) != 3 {
		t.Fatalf("unexpected role enum: %+v", roleEnum)
	}
	if roleEnum.Cases[0].Name != "admin" || roleEnum.Cases[1].Name != "viewer" || roleEnum.Cases[2].Name != "editor" {
		t.Errorf("enum cases mismatch: %+v", roleEnum.Cases)
	}

	// TypeDefs
	if len(iface.TypeDefs) != 2 {
		t.Fatalf("expected 2 type defs, got %d", len(iface.TypeDefs))
	}
	if iface.TypeDefs[0].Name != "user-alias" || iface.TypeDefs[0].Type.Kind != KindNamed {
		t.Errorf("unexpected typedef 0: %+v", iface.TypeDefs[0])
	}
	if iface.TypeDefs[1].Name != "id-alias" || iface.TypeDefs[1].Type.Kind != KindPrimitive {
		t.Errorf("unexpected typedef 1: %+v", iface.TypeDefs[1])
	}
}

func TestParser_WorldImportsAndExports(t *testing.T) {
	input := `
	package app:system;

	interface logger {
		log: func(msg: string);
	}

	interface handler {
		handle: func();
	}

	world app-world {
		import logger;
		import config;
		export handler;
		export metrics;
	}
	`
	pkg, err := ParseContent(input)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if len(pkg.Worlds) != 1 {
		t.Fatalf("expected 1 world, got %d", len(pkg.Worlds))
	}

	w := pkg.Worlds[0]
	if w.Name != "app-world" {
		t.Errorf("expected world name app-world, got %q", w.Name)
	}
	if len(w.Imports) != 2 || w.Imports[0] != "logger" || w.Imports[1] != "config" {
		t.Errorf("unexpected imports: %v", w.Imports)
	}
	if len(w.Exports) != 2 || w.Exports[0] != "handler" || w.Exports[1] != "metrics" {
		t.Errorf("unexpected exports: %v", w.Exports)
	}
}

func TestParser_TypeRefStringRepresentation(t *testing.T) {
	var nilRef *TypeRef
	if nilRef.String() != "unit" {
		t.Errorf("expected 'unit' for nil TypeRef, got %q", nilRef.String())
	}

	listRef := &TypeRef{Kind: KindList, Name: "list", TypeArgs: []*TypeRef{{Kind: KindPrimitive, Name: "s32"}}}
	if listRef.String() != "list<s32>" {
		t.Errorf("expected 'list<s32>', got %q", listRef.String())
	}

	optRef := &TypeRef{Kind: KindOption, Name: "option", TypeArgs: []*TypeRef{{Kind: KindPrimitive, Name: "string"}}}
	if optRef.String() != "option<string>" {
		t.Errorf("expected 'option<string>', got %q", optRef.String())
	}

	res1Ref := &TypeRef{Kind: KindResult, Name: "result", TypeArgs: []*TypeRef{{Kind: KindPrimitive, Name: "s32"}}}
	if res1Ref.String() != "result<s32>" {
		t.Errorf("expected 'result<s32>', got %q", res1Ref.String())
	}

	res2Ref := &TypeRef{Kind: KindResult, Name: "result", TypeArgs: []*TypeRef{
		{Kind: KindPrimitive, Name: "s32"},
		{Kind: KindPrimitive, Name: "string"},
	}}
	if res2Ref.String() != "result<s32, string>" {
		t.Errorf("expected 'result<s32, string>', got %q", res2Ref.String())
	}

	tupRef := &TypeRef{Kind: KindTuple, Name: "tuple", TypeArgs: []*TypeRef{
		{Kind: KindPrimitive, Name: "u8"},
		{Kind: KindPrimitive, Name: "u16"},
	}}
	if tupRef.String() != "tuple<u8, u16>" {
		t.Errorf("expected 'tuple<u8, u16>', got %q", tupRef.String())
	}
}

func TestParser_ErrorCases(t *testing.T) {
	errorInputs := []struct {
		name  string
		input string
	}{
		{"missing interface name", "interface { }"},
		{"missing opening brace", "interface api }"},
		{"missing colon after func name", "interface api { my-fn func(); }"},
		{"missing func keyword", "interface api { my-fn: notfunc(); }"},
		{"unclosed param list", "interface api { my-fn: func(x: s32 ; }"},
		{"unclosed record brace", "interface api { record user { id: u64 }"},
		{"missing equals in type alias", "interface api { type my-type s32; }"},
	}

	for _, tt := range errorInputs {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseContent(tt.input)
			if err == nil {
				t.Errorf("expected error for %q, but got nil", tt.input)
			}
		})
	}
}

func TestParser_ParseFileAndPathErrors(t *testing.T) {
	// Parse non-existent file
	_, err := ParseFile("/non/existent/path/test.wit")
	if err == nil {
		t.Errorf("expected error for non-existent file, got nil")
	}

	// ParsePath on non-existent path
	_, err = ParsePath("/non/existent/path")
	if err == nil {
		t.Errorf("expected error for non-existent path, got nil")
	}

	// ParsePath on directory with multiple WIT files
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "pkg.wit")
	f2 := filepath.Join(tmpDir, "iface2.wit")

	c1 := `
	package multi:test@1.0.0;
	interface one {
		fn1: func();
	}
	`
	c2 := `
	interface two {
		fn2: func();
	}
	world multi-world {
		export one;
		export two;
	}
	`
	if err := os.WriteFile(f1, []byte(c1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte(c2), 0644); err != nil {
		t.Fatal(err)
	}

	pkg, err := ParsePath(tmpDir)
	if err != nil {
		t.Fatalf("ParsePath failed on multi-file directory: %v", err)
	}
	if pkg.FullName() != "multi:test" {
		t.Errorf("expected package full name multi:test, got %q", pkg.FullName())
	}
	if pkg.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %q", pkg.Version)
	}
	if len(pkg.Interfaces) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(pkg.Interfaces))
	}
	if len(pkg.Worlds) != 1 {
		t.Errorf("expected 1 world, got %d", len(pkg.Worlds))
	}

	// Single file directly to ParsePath
	pkgSingle, err := ParsePath(f1)
	if err != nil {
		t.Fatalf("ParsePath failed on single file: %v", err)
	}
	if len(pkgSingle.Interfaces) != 1 || pkgSingle.Interfaces[0].Name != "one" {
		t.Errorf("unexpected single file package: %+v", pkgSingle)
	}

	// Malformed file in directory
	badFile := filepath.Join(tmpDir, "bad.wit")
	if err := os.WriteFile(badFile, []byte("interface broken {"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = ParsePath(tmpDir)
	if err == nil {
		t.Errorf("expected error when directory contains malformed WIT, got nil")
	}
}
