package generate

import (
	"testing"
	"tools/please_wit/ast"
)

func TestGoGenerator_MapWitType(t *testing.T) {
	gen := &GoGenerator{}

	tests := []struct {
		name    string
		typeRef *ast.TypeRef
		want    string
	}{
		{name: "nil", typeRef: nil, want: ""},
		{name: "s8", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s8"}, want: "int8"},
		{name: "s16", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s16"}, want: "int16"},
		{name: "s32", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s32"}, want: "int32"},
		{name: "s64", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s64"}, want: "int64"},
		{name: "u8", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "u8"}, want: "uint8"},
		{name: "u16", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "u16"}, want: "uint16"},
		{name: "u32", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "u32"}, want: "uint32"},
		{name: "u64", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "u64"}, want: "uint64"},
		{name: "f32", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "f32"}, want: "float32"},
		{name: "f64", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "f64"}, want: "float64"},
		{name: "bool", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "bool"}, want: "bool"},
		{name: "string", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "string"}, want: "string"},
		{name: "char", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "char"}, want: "rune"},
		{name: "unit", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "unit"}, want: ""},
		{name: "underscore", typeRef: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "_"}, want: ""},
		{name: "list of s32", typeRef: &ast.TypeRef{Kind: ast.KindList, TypeArgs: []*ast.TypeRef{{Kind: ast.KindPrimitive, Name: "s32"}}}, want: "[]int32"},
		{name: "empty list", typeRef: &ast.TypeRef{Kind: ast.KindList}, want: "[]any"},
		{name: "option of string", typeRef: &ast.TypeRef{Kind: ast.KindOption, TypeArgs: []*ast.TypeRef{{Kind: ast.KindPrimitive, Name: "string"}}}, want: "*string"},
		{name: "empty option", typeRef: &ast.TypeRef{Kind: ast.KindOption}, want: "*any"},
		{name: "result of s32", typeRef: &ast.TypeRef{Kind: ast.KindResult, TypeArgs: []*ast.TypeRef{{Kind: ast.KindPrimitive, Name: "s32"}}}, want: "(int32, error)"},
		{name: "result of unit", typeRef: &ast.TypeRef{Kind: ast.KindResult, TypeArgs: []*ast.TypeRef{{Kind: ast.KindPrimitive, Name: "unit"}}}, want: "error"},
		{name: "empty result", typeRef: &ast.TypeRef{Kind: ast.KindResult}, want: "error"},
		{name: "tuple of s32 and string", typeRef: &ast.TypeRef{Kind: ast.KindTuple, TypeArgs: []*ast.TypeRef{{Kind: ast.KindPrimitive, Name: "s32"}, {Kind: ast.KindPrimitive, Name: "string"}}}, want: "struct{int32; string}"},
		{name: "named type", typeRef: &ast.TypeRef{Kind: ast.KindNamed, Name: "my-point"}, want: "MyPoint"},
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

func TestGoGenerator_ResolvePackage(t *testing.T) {
	gen := &GoGenerator{}

	tests := []struct {
		name          string
		targetPackage string
		pkg           *ast.Package
		want          string
	}{
		{
			name:          "explicit target package",
			targetPackage: "my:custom-pkg",
			pkg:           &ast.Package{},
			want:          "custom_pkg",
		},
		{
			name:          "package namespace and name",
			targetPackage: "",
			pkg:           &ast.Package{Namespace: "test", Name: "structures"},
			want:          "structures",
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
			got := gen.resolvePackage(tc.pkg, tc.targetPackage)
			if got != tc.want {
				t.Errorf("name: %s, want: %q, got: %q", tc.name, tc.want, got)
			}
		})
	}
}

func TestGoGeneratorDirect(t *testing.T) {
	gen := &GoGenerator{}

	t.Run("generator name", func(t *testing.T) {
		want := "go"
		got := gen.Name()
		if got != want {
			t.Errorf("name: generator name, want: %q, got: %q", want, got)
		}
	})

	tests := []struct {
		name        string
		pkg         *ast.Package
		opts        Options
		baseName    string
		wantContent string
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
			wantContent: `// Code generated by please_wit from WIT AST. DO NOT EDIT.
package api

type Greeter interface {
	SayHello(name string) string
}

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
			wantContent: `// Code generated by please_wit from WIT AST. DO NOT EDIT.
package types

type User struct {
	Id int64
	Name string
}

type Status int

const (
	Status_Active Status = iota
	Status_Inactive
)

type UserId = int64

type DataService interface {
}

`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files, err := gen.Generate(tc.pkg, tc.opts, tc.baseName)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			if len(files) != 1 {
				t.Fatalf("expected 1 file, got %d", len(files))
			}
			got := files[0].Content
			if got != tc.wantContent {
				t.Errorf("name: %s, want: %q, got: %q", tc.name, tc.wantContent, got)
			}
		})
	}
}
