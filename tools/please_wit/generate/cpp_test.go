package generate

import (
	"testing"
	"tools/please_wit/ast"
)

func TestCppGenerator_MapWitType(t *testing.T) {
	gen := &CppGenerator{}

	tests := []struct {
		name    string
		typeRef *ast.TypeRef
		want    string
	}{
		{name: "nil", typeRef: nil, want: "void"},
		{name: "s8", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s8"}, want: "int8_t"},
		{name: "s16", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s16"}, want: "int16_t"},
		{name: "s32", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s32"}, want: "int32_t"},
		{name: "s64", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s64"}, want: "int64_t"},
		{name: "u8", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "u8"}, want: "uint8_t"},
		{name: "u16", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "u16"}, want: "uint16_t"},
		{name: "u32", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "u32"}, want: "uint32_t"},
		{name: "u64", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "u64"}, want: "uint64_t"},
		{name: "f32", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "f32"}, want: "float"},
		{name: "f64", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "f64"}, want: "double"},
		{name: "bool", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "bool"}, want: "bool"},
		{name: "string", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "string"}, want: "std::string"},
		{name: "char", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "char"}, want: "char32_t"},
		{name: "unit", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "unit"}, want: "void"},
		{name: "underscore", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "_"}, want: "void"},
		{name: "list of s32", typeRef: &ast.TypeRef{Kind: ast.KindList, TypeArgs: []*ast.TypeRef{{Kind: ast.KindPrimitive, Name: "s32"}}}, want: "std::vector<int32_t>"},
		{name: "empty list", typeRef: &ast.TypeRef{Kind: ast.KindList}, want: "std::vector<void>"},
		{name: "option of string", typeRef: &ast.TypeRef{Kind: ast.KindOption, TypeArgs: []*ast.TypeRef{{Kind: ast.KindPrimitive, Name: "string"}}}, want: "std::optional<std::string>"},
		{name: "empty option", typeRef: &ast.TypeRef{Kind: ast.KindOption}, want: "std::optional<void>"},
		{name: "result of s32", typeRef: &ast.TypeRef{Kind: ast.KindResult, TypeArgs: []*ast.TypeRef{{Kind: ast.KindPrimitive, Name: "s32"}}}, want: "std::optional<int32_t>"},
		{name: "result of void", typeRef: &ast.TypeRef{Kind: ast.KindResult, TypeArgs: []*ast.TypeRef{{Kind: ast.KindPrimitive, Name: "unit"}}}, want: "bool"},
		{name: "empty result", typeRef: &ast.TypeRef{Kind: ast.KindResult}, want: "bool"},
		{name: "tuple of s32 and string", typeRef: &ast.TypeRef{Kind: ast.KindTuple, TypeArgs: []*ast.TypeRef{{Kind: ast.KindPrimitive, Name: "s32"}, {Kind: ast.KindPrimitive, Name: "string"}}}, want: "std::tuple<int32_t, std::string>"},
		{name: "named type", typeRef: &ast.TypeRef{Kind: ast.KindNamed, Name: "my-point"}, want: "MyPoint"},
		{name: "unknown primitive", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "unknown-type"}, want: "UnknownType"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := gen.MapWitType(tc.typeRef)
			if got != tc.want {
				t.Errorf("name: %s, want: %q, got: %q", tc.name, tc.want, got)
			}
		})
	}
}

func TestCppGenerator_CreateNamespace(t *testing.T) {
	gen := &CppGenerator{}

	tests := []struct {
		name          string
		targetPackage string
		pkg           *ast.Package
		want          string
	}{
		{
			name:          "explicit target package",
			targetPackage: "my::custom-pkg",
			pkg:           &ast.Package{},
			want:          "my__custom_pkg",
		},
		{
			name:          "package namespace and name",
			targetPackage: "",
			pkg:           &ast.Package{Namespace: "test", Name: "structures"},
			want:          "test_structures",
		},
		{
			name:          "package name only",
			targetPackage: "",
			pkg:           &ast.Package{Name: "structures"},
			want:          "structures",
		},
		{
			name:          "fallback default wit",
			targetPackage: "",
			pkg:           &ast.Package{},
			want:          "wit",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := gen.createNamespace(tc.targetPackage, tc.pkg)
			if got != tc.want {
				t.Errorf("name: %s, want: %q, got: %q", tc.name, tc.want, got)
			}
		})
	}
}

func TestCppGeneratorDirect(t *testing.T) {
	gen := &CppGenerator{}

	t.Run("generator name", func(t *testing.T) {
		want := "cpp"
		got := gen.Name()
		if got != want {
			t.Errorf("name: generator name, want: %q, got: %q", want, got)
		}
	})

	tests := []struct {
		name       string
		pkg        *ast.Package
		opts       Options
		baseName   string
		wantHeader string
		wantSource string
	}{
		{
			name: "basic interface with params and return",
			pkg: &ast.Package{
				Namespace: "demo",
				Name:      "api",
				Interfaces: []ast.Interface{
					{
						Name: "greeter",
						Functions: []ast.Function{
							{
								Name: "say-hello",
								Params: []ast.Param{
									{Name: "name", Type: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "string"}},
								},
								Results: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "string"},
							},
						},
					},
				},
			},
			opts:     Options{},
			baseName: "api",
			wantHeader: `// Auto-generated by please_wit from WIT AST. DO NOT EDIT.
#pragma once

#include <cstdint>
#include <string>
#include <vector>
#include <optional>
#include <tuple>

namespace demo_api {

class Greeter {
public:
    virtual ~Greeter() = default;
    virtual std::string sayHello(const std::string& name) = 0;
};

} // namespace demo_api
`,
			wantSource: `// Auto-generated by please_wit from WIT AST. DO NOT EDIT.
#include "api.h"
`,
		},
		{
			name: "records, enums, and type aliases",
			pkg: &ast.Package{
				Name: "types",
				Interfaces: []ast.Interface{
					{
						Name: "data-service",
						Records: []ast.Record{
							{
								Name: "user",
								Fields: []ast.Field{
									{Name: "id", Type: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s64"}},
									{Name: "name", Type: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "string"}},
								},
							},
						},
						Enums: []ast.Enum{
							{
								Name:  "status",
								Cases: []ast.EnumCase{{Name: "active"}, {Name: "inactive"}},
							},
						},
						TypeDefs: []ast.TypeDef{
							{Name: "user-id", Type: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s64"}},
						},
					},
				},
			},
			opts:     Options{},
			baseName: "types",
			wantHeader: `// Auto-generated by please_wit from WIT AST. DO NOT EDIT.
#pragma once

#include <cstdint>
#include <string>
#include <vector>
#include <optional>
#include <tuple>

namespace types {

struct User {
    int64_t id;
    std::string name;
};

enum class Status {
    Active,
    Inactive,
};

using UserId = int64_t;

class DataService {
public:
    virtual ~DataService() = default;
};

} // namespace types
`,
			wantSource: `// Auto-generated by please_wit from WIT AST. DO NOT EDIT.
#include "types.h"
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files, err := gen.Generate(tc.pkg, tc.opts, tc.baseName)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			if len(files) != 2 {
				t.Fatalf("expected 2 files (header and source), got %d", len(files))
			}
			got := files[0].Content
			if got != tc.wantHeader {
				t.Errorf("name: %s (header), want: %q, got: %q", tc.name, tc.wantHeader, got)
			}
			got = files[1].Content
			if got != tc.wantSource {
				t.Errorf("name: %s (source), want: %q, got: %q", tc.name, tc.wantSource, got)
			}
		})
	}
}
