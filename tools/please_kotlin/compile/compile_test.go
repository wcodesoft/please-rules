package compile

import (
	"reflect"
	"testing"
)

func TestBuildKotlincArgs(t *testing.T) {
	opts := Options{
		JvmTarget:  "21",
		ModuleName: "my_module",
		Deps:       []string{"lib1.jar", "lib2.jar"},
		Flags:      []string{"-verbose"},
		Srcs:       []string{"a.kt", "b.kt"},
	}

	args := BuildKotlincArgs("/tmp/classes", opts)

	expectedPrefix := []string{
		"-d", "/tmp/classes",
		"-jvm-target", "21",
		"-module-name", "my_module",
	}

	for i, exp := range expectedPrefix {
		if args[i] != exp {
			t.Errorf("args[%d] = %q, want %q", i, args[i], exp)
		}
	}

	// Verify sources are at the end
	if args[len(args)-2] != "a.kt" || args[len(args)-1] != "b.kt" {
		t.Errorf("sources not at end: %v", args[len(args)-2:])
	}
}

func TestExpandCommaSeparated(t *testing.T) {
	input := []string{"a.kt, b.kt", "c.kt", "  d.kt  "}
	got := ExpandCommaSeparated(input)
	want := []string{"a.kt", "b.kt", "c.kt", "d.kt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRunValidation(t *testing.T) {
	if err := Run(Options{}); err == nil {
		t.Errorf("expected error for empty options")
	}
	if err := Run(Options{Out: "out.jar"}); err == nil {
		t.Errorf("expected error for missing srcs")
	}
}
