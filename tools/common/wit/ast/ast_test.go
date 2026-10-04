package ast

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSimpleWit(t *testing.T) {
	input := `
// Package comment
package test:structures@0.1.0;

interface two-sum {
    solve: func(nums: list<s32>, target: s32) -> list<s32>;
    reset: func();
}

world two-sum-world {
    export two-sum;
}
`
	pkg, err := ParseContent(input)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if pkg.Namespace != "test" {
		t.Errorf("expected namespace 'test', got %q", pkg.Namespace)
	}
	if pkg.Name != "structures" {
		t.Errorf("expected name 'structures', got %q", pkg.Name)
	}
	if pkg.Version != "0.1.0" {
		t.Errorf("expected version '0.1.0', got %q", pkg.Version)
	}

	if len(pkg.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(pkg.Interfaces))
	}
	iface := pkg.Interfaces[0]
	if iface.Name != "two-sum" {
		t.Errorf("expected interface 'two-sum', got %q", iface.Name)
	}

	if len(iface.Functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(iface.Functions))
	}

	fn1 := iface.Functions[0]
	if fn1.Name != "solve" {
		t.Errorf("expected fn 'solve', got %q", fn1.Name)
	}
	if len(fn1.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(fn1.Params))
	}
	if fn1.Params[0].Name != "nums" || fn1.Params[0].Type.Kind != KindList {
		t.Errorf("unexpected param 0: %+v", fn1.Params[0])
	}
	if fn1.Params[1].Name != "target" || fn1.Params[1].Type.Name != "s32" {
		t.Errorf("unexpected param 1: %+v", fn1.Params[1])
	}
	if fn1.Results == nil || fn1.Results.Kind != KindList {
		t.Errorf("unexpected results: %+v", fn1.Results)
	}

	fn2 := iface.Functions[1]
	if fn2.Name != "reset" || fn2.Results != nil {
		t.Errorf("unexpected fn2: %+v", fn2)
	}

	if len(pkg.Worlds) != 1 {
		t.Fatalf("expected 1 world, got %d", len(pkg.Worlds))
	}
	world := pkg.Worlds[0]
	if world.Name != "two-sum-world" {
		t.Errorf("expected world 'two-sum-world', got %q", world.Name)
	}
	if len(world.Exports) != 1 || world.Exports[0] != "two-sum" {
		t.Errorf("unexpected exports: %v", world.Exports)
	}
}

func TestParseRecordsAndEnums(t *testing.T) {
	input := `
package test:types;

interface data-types {
    /* Multi line comment
       about point
    */
    record point {
        x: f64,
        y: f64,
    }

    enum color {
        red,
        green,
        blue,
    }

    type point-alias = point;

    get-color: func(p: point) -> option<color>;
    calc: func(a: s32) -> result<s32, string>;
}
`
	pkg, err := ParseContent(input)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if len(pkg.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(pkg.Interfaces))
	}
	iface := pkg.Interfaces[0]

	if len(iface.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(iface.Records))
	}
	rec := iface.Records[0]
	if rec.Name != "point" || len(rec.Fields) != 2 {
		t.Errorf("unexpected record: %+v", rec)
	}

	if len(iface.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(iface.Enums))
	}
	enm := iface.Enums[0]
	if enm.Name != "color" || len(enm.Cases) != 3 {
		t.Errorf("unexpected enum: %+v", enm)
	}

	if len(iface.TypeDefs) != 1 {
		t.Fatalf("expected 1 type def, got %d", len(iface.TypeDefs))
	}

	if len(iface.Functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(iface.Functions))
	}

	fn1 := iface.Functions[0]
	if fn1.Results == nil || fn1.Results.Kind != KindOption {
		t.Errorf("expected option result, got %+v", fn1.Results)
	}

	fn2 := iface.Functions[1]
	if fn2.Results == nil || fn2.Results.Kind != KindResult {
		t.Errorf("expected result result, got %+v", fn2.Results)
	}
}

func TestParsePath(t *testing.T) {
	tmpDir := t.TempDir()
	witFile := filepath.Join(tmpDir, "test.wit")
	content := `
package test:structures;

interface disjoint-set {
    find: func(x: s32) -> s32;
}
`
	if err := os.WriteFile(witFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pkg, err := ParsePath(tmpDir)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}
	if pkg.FullName() != "test:structures" {
		t.Errorf("expected test:structures, got %q", pkg.FullName())
	}
	if len(pkg.Interfaces) != 1 || pkg.Interfaces[0].Name != "disjoint-set" {
		t.Errorf("unexpected interfaces: %+v", pkg.Interfaces)
	}
}
