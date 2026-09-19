package ast

import (
	"fmt"
	"strings"
)

// TypeKind distinguishes between primitives, compound types, and user references.
type TypeKind int

const (
	KindPrimitive TypeKind = iota
	KindList
	KindOption
	KindResult
	KindTuple
	KindNamed
)

// TypeRef represents a type reference in WIT.
type TypeRef struct {
	Kind     TypeKind
	Name     string     // e.g. "s32", "string", or user-defined type name
	TypeArgs []*TypeRef // For list<T>, option<T>, result<T, E>, tuple<...>
}

func (t *TypeRef) String() string {
	if t == nil {
		return "unit"
	}
	switch t.Kind {
	case KindPrimitive, KindNamed:
		return t.Name
	case KindList:
		if len(t.TypeArgs) > 0 {
			return fmt.Sprintf("list<%s>", t.TypeArgs[0].String())
		}
		return "list"
	case KindOption:
		if len(t.TypeArgs) > 0 {
			return fmt.Sprintf("option<%s>", t.TypeArgs[0].String())
		}
		return "option"
	case KindResult:
		if len(t.TypeArgs) == 2 {
			return fmt.Sprintf("result<%s, %s>", t.TypeArgs[0].String(), t.TypeArgs[1].String())
		} else if len(t.TypeArgs) == 1 {
			return fmt.Sprintf("result<%s>", t.TypeArgs[0].String())
		}
		return "result"
	case KindTuple:
		var inner []string
		for _, arg := range t.TypeArgs {
			inner = append(inner, arg.String())
		}
		return fmt.Sprintf("tuple<%s>", strings.Join(inner, ", "))
	}
	return t.Name
}

// Param represents a parameter in a WIT function.
type Param struct {
	Name string
	Type *TypeRef
}

// Function represents a function declared in a WIT interface.
type Function struct {
	Name    string
	Doc     string
	Params  []Param
	Results *TypeRef // nil means void / unit
}

// Field represents a field in a WIT record.
type Field struct {
	Name string
	Type *TypeRef
}

// Record represents a record structure declared in a WIT interface.
type Record struct {
	Name   string
	Doc    string
	Fields []Field
}

// EnumCase represents a single case in an enum.
type EnumCase struct {
	Name string
}

// Enum represents an enum declared in a WIT interface.
type Enum struct {
	Name  string
	Doc   string
	Cases []EnumCase
}

// TypeDef represents a type alias declared in WIT.
type TypeDef struct {
	Name string
	Type *TypeRef
}

// Interface represents an interface block in WIT.
type Interface struct {
	Name      string
	Doc       string
	Functions []Function
	Records   []Record
	Enums     []Enum
	TypeDefs  []TypeDef
}

// World represents a world definition in WIT.
type World struct {
	Name    string
	Doc     string
	Imports []string
	Exports []string
}

// Package represents a WIT package containing interfaces and worlds.
type Package struct {
	Namespace  string
	Name       string
	Version    string
	Interfaces []Interface
	Worlds     []World
}

// FullName returns the canonical package identifier (e.g. "namespace:name").
func (p *Package) FullName() string {
	if p.Namespace != "" && p.Name != "" {
		return fmt.Sprintf("%s:%s", p.Namespace, p.Name)
	}
	if p.Name != "" {
		return p.Name
	}
	return p.Namespace
}
