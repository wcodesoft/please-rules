package css

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func names(defs []VarDef) []string {
	var out []string
	for _, d := range defs {
		out = append(out, d.Name)
	}
	return out
}

func TestParseVariablesDefinitions(t *testing.T) {
	src := `
:root { --a: 1; --b: var(--a); }
[data-theme="dark"] { --c: 2; }
.card { --d: 3; color: red; }
@media (max-width: 600px) { .card { --e: 4; } }
@property --f { syntax: "<color>"; inherits: false; initial-value: red; }
.parent { .child { --g: 5; } }
`
	vars, err := ParseVariables("a.css", src)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--a", "--b", "--c", "--d", "--e", "--f", "--g"}
	got := names(vars.Defs)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("definitions = %v, want %v", got, want)
	}
}

func TestParseVariablesUses(t *testing.T) {
	tests := []struct {
		name string
		css  string
		want []VarUse // Name, Line, Fallback
	}{
		{
			name: "a plain use",
			css:  ".a { color: var(--x); }",
			want: []VarUse{{Name: "--x", Line: 1}},
		},
		{
			name: "with a fallback",
			css:  ".a { color: var(--x, red); }",
			want: []VarUse{{Name: "--x", Line: 1, Fallback: true}},
		},
		{
			name: "an empty fallback is a fallback",
			css:  ".a { color: var(--x,); }",
			want: []VarUse{{Name: "--x", Line: 1, Fallback: true}},
		},
		{
			name: "spaces and newlines inside the call",
			css:  ".a {\n  color: var(\n    --x\n  );\n}",
			want: []VarUse{{Name: "--x", Line: 2}},
		},
		{
			name: "a use inside another's fallback is its own use",
			css:  ".a { color: var(--x, var(--y)); }",
			want: []VarUse{{Name: "--x", Line: 1, Fallback: true}, {Name: "--y", Line: 1}},
		},
		{
			name: "inside calc and a shorthand",
			css:  ".a { margin: calc(var(--s) * 2) var(--t); }",
			want: []VarUse{{Name: "--s", Line: 1}, {Name: "--t", Line: 1}},
		},
		{
			name: "in a custom property value",
			css:  ":root { --b: var(--a); }",
			want: []VarUse{{Name: "--a", Line: 1}},
		},
		{
			name: "in @keyframes",
			css:  "@keyframes k {\n  from { opacity: var(--o); }\n  to { opacity: 1; }\n}",
			want: []VarUse{{Name: "--o", Line: 2}},
		},
		{
			name: "in @media and @layer",
			css:  "@layer base { @media (min-width: 1px) { .a { gap: var(--g); } } }",
			want: []VarUse{{Name: "--g", Line: 1}},
		},
		{
			name: "in a nested rule",
			css:  ".a { .b { color: var(--n); } }",
			want: []VarUse{{Name: "--n", Line: 1}},
		},
		{
			name: "on the line of the use in a multi-line value",
			css:  ".a {\n  box-shadow: 0 0 1px red,\n    0 0 2px var(--s);\n}",
			want: []VarUse{{Name: "--s", Line: 3}},
		},
		{
			name: "not in a comment",
			css:  ".a { color: red; /* var(--gone) */ }",
		},
		{
			name: "not in a string",
			css:  `.a { content: "var(--gone)"; }`,
		},
		{
			name: "not the tail of another function name",
			css:  ".a { width: myvar(--gone); }",
		},
		{
			name: "not a var() without a custom property",
			css:  ".a { color: var(notcustom); }",
		},
		{
			name: "names are case sensitive and keep their case",
			css:  ".a { color: var(--Brand-Color); }",
			want: []VarUse{{Name: "--Brand-Color", Line: 1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars, err := ParseVariables("a.css", tt.css)
			if err != nil {
				t.Fatal(err)
			}
			for i := range tt.want {
				tt.want[i].File = "a.css"
			}
			if len(vars.Uses) != len(tt.want) || (len(tt.want) > 0 && !reflect.DeepEqual(vars.Uses, tt.want)) {
				t.Errorf("uses = %+v, want %+v", vars.Uses, tt.want)
			}
		})
	}
}

func TestParseVariablesStillReportsMalformedCSS(t *testing.T) {
	for name, css := range map[string]string{
		"unclosed keyframes": "@keyframes k { from { opacity: var(--o);",
		"unclosed font-face": "@font-face { font-family: x;",
		"unclosed nested":    ".a { .b { color: red; }",
	} {
		if _, err := ParseVariables("a.css", css); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestFindUndefinedVars(t *testing.T) {
	vars := Variables{
		Defs: []VarDef{{Name: "--a"}, {Name: "--b"}},
		Uses: []VarUse{
			{Name: "--a"}, // defined
			{Name: "--missing", File: "x.css", Line: 3}, // undefined
			{Name: "--missing", File: "x.css", Line: 9},
			{Name: "--soft", Fallback: true},            // undefined but has a fallback
			{Name: "--chart-1", File: "y.css", Line: 1}, // matched by an allowed prefix
			{Name: "--runtime", File: "y.css", Line: 2}, // allowed exactly
			{Name: "--Missing", File: "x.css", Line: 5}, // case differs from --missing
		},
	}
	got := FindUndefinedVars(vars, []string{"--chart-*", "--runtime"})
	var gotNames []string
	for _, v := range got {
		gotNames = append(gotNames, v.Name)
	}
	if want := []string{"--Missing", "--missing"}; !reflect.DeepEqual(gotNames, want) {
		t.Fatalf("undefined = %v, want %v", gotNames, want)
	}
	if n := len(got[1].Uses); n != 2 {
		t.Errorf("--missing has %d uses, want 2", n)
	}
	if len(FindUndefinedVars(Variables{}, nil)) != 0 {
		t.Error("no variables must give no findings")
	}
}

func writeCSS(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCheckVariablesAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	tokens := writeCSS(t, dir, "tokens.css", ":root { --brand: #fff; --space: 1rem; }\n")
	button := writeCSS(t, dir, "button.css", ".btn {\n  color: var(--brand);\n  padding: var(--space);\n}\n")
	bad := writeCSS(t, dir, "bad.css", ".x {\n  font-weight: var(--weight-medium);\n}\n.y {\n  font-weight: var(--weight-medium);\n  gap: var(--gap, 1rem);\n  top: var(--offset);\n}\n")

	// A variable defined in a dependency's file is defined for the bundle.
	if err := CheckVariables([]string{tokens, button}, nil); err != nil {
		t.Fatalf("variables defined in another file were reported: %v", err)
	}
	// The same file alone does not define them.
	if err := CheckVariables([]string{button}, nil); err == nil {
		t.Error("a use whose definition is not in the bundle was accepted")
	}

	err := CheckVariables([]string{tokens, button, bad}, nil)
	if err == nil {
		t.Fatal("undefined variables were accepted")
	}
	msg := err.Error()
	for _, want := range []string{
		"2 custom properties are used with var()",
		"--weight-medium: 2 uses without a fallback",
		bad + ":2",
		bad + ":5",
		"--offset: 1 use without a fallback",
		bad + ":7",
		"allow_undefined_vars",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "--gap") {
		t.Errorf("a use with a fallback was reported:\n%s", msg)
	}

	// Allowed names, exactly or by prefix.
	if err := CheckVariables([]string{tokens, button, bad}, []string{"--weight-medium", "--off*"}); err != nil {
		t.Errorf("allowed variables were reported: %v", err)
	}
}

func TestCheckVariablesListsAtMostFivePlaces(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	for i := 0; i < 8; i++ {
		sb.WriteString(".a { color: var(--many); }\n")
	}
	f := writeCSS(t, dir, "many.css", sb.String())
	err := CheckVariables([]string{f}, nil)
	if err == nil || !strings.Contains(err.Error(), "--many: 8 uses") || !strings.Contains(err.Error(), "and 3 more") {
		t.Errorf("long lists are not shortened: %v", err)
	}
}

func TestCheckVariablesRejectsBadAllowEntries(t *testing.T) {
	dir := t.TempDir()
	f := writeCSS(t, dir, "a.css", ".a { color: red; }\n")
	for _, entry := range []string{"brand", "-brand", "--", "--*", ""} {
		if err := CheckVariables([]string{f}, []string{entry}); err == nil {
			t.Errorf("allow entry %q was accepted", entry)
		}
	}
}

func TestCheckVariablesReportsUnreadableAndMalformedFiles(t *testing.T) {
	dir := t.TempDir()
	if err := CheckVariables([]string{filepath.Join(dir, "missing.css")}, nil); err == nil {
		t.Error("a missing file was accepted")
	}
	bad := writeCSS(t, dir, "bad.css", ".a { color: red;")
	if err := CheckVariables([]string{bad}, nil); err == nil {
		t.Error("a malformed file was accepted")
	}
}
