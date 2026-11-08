package css

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDefinitionsExtractsSelectorsKeyframesAndVariables(t *testing.T) {
	src := `
/* Header comment with .ignored { color: red; } */
:root {
	--primary-color: #1e40af;
	--spacing-md: 16px;
	color: var(--primary-color);
}

.btn, .btn-primary {
	padding: var(--spacing-md);
	content: "{ not a block }";
}

.card > .header {
	font-weight: bold;
}

@keyframes fade-in {
	from { opacity: 0; }
	to { opacity: 1; }
}

@media (max-width: 600px) {
	.btn {
		padding: 8px;
	}
	:root {
		--spacing-md: 8px;
	}
}
`
	defs, err := ParseDefinitions("ui/button.css", src)
	if err != nil {
		t.Fatalf("ParseDefinitions failed: %v", err)
	}

	gotKeys := make(map[string]bool)
	for _, d := range defs {
		gotKeys[d.Key()] = true
	}

	expected := []string{
		"selector||:root",
		"variable||--primary-color",
		"variable||--spacing-md",
		"selector||.btn",
		"selector||.btn-primary",
		"selector||.card>.header",
		"keyframes||fade-in",
		"selector|@media (max-width:600px)|.btn",
		"selector|@media (max-width:600px)|:root",
		"variable|@media (max-width:600px)|--spacing-md",
	}

	for _, want := range expected {
		if !gotKeys[want] {
			t.Errorf("missing expected definition key %q in %v", want, defs)
		}
	}
}

func TestParseDefinitionsRejectsSyntaxErrors(t *testing.T) {
	cases := []struct {
		name string
		css  string
		want string
	}{
		{
			name: "unclosed brace",
			css:  ".card { color: red;",
			want: "unexpected end of CSS file",
		},
		{
			name: "unexpected closing brace",
			css:  ".card { color: red; } }",
			want: "unexpected '}'",
		},
		{
			name: "unterminated string",
			css:  ".card { content: \"hello\nworld\"; }",
			want: "unterminated string literal",
		},
		{
			name: "unclosed comment",
			css:  "/* unclosed comment\n.card { color: red; }",
			want: "unclosed CSS comment",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseDefinitions("broken.css", tc.css)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestFindCollisionsAllowsSameFileAndDifferentMediaScopes(t *testing.T) {
	fileA := `
:root { --accent: blue; }
.btn { color: blue; }
.btn { background: white; }
@media (max-width: 600px) {
	.btn { color: red; }
}
`
	fileB := `
@media (min-width: 601px) {
	.btn { color: green; }
}
.card { padding: 12px; }
`
	defsA, err := ParseDefinitions("a.css", fileA)
	if err != nil {
		t.Fatal(err)
	}
	defsB, err := ParseDefinitions("b.css", fileB)
	if err != nil {
		t.Fatal(err)
	}

	collisions := FindCollisions(append(defsA, defsB...))
	if len(collisions) != 0 {
		t.Fatalf("expected 0 collisions, got %v", collisions)
	}
}

func TestCheckFilesDetectsCrossFileSelectorKeyframesAndVariableCollisions(t *testing.T) {
	dir := t.TempDir()
	buttonCSS := filepath.Join(dir, "button.css")
	modalCSS := filepath.Join(dir, "modal.css")

	err := os.WriteFile(buttonCSS, []byte(`:root {
  --brand: #111;
}
.title {
  font-size: 16px;
}
@keyframes pulse {
  from { opacity: 0.5; }
  to { opacity: 1; }
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(modalCSS, []byte(`:root {
  --brand: #222;
}
.title {
  font-size: 20px;
}
@keyframes pulse {
  from { opacity: 0; }
  to { opacity: 1; }
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	checkErr := CheckFiles([]string{buttonCSS, modalCSS})
	if checkErr == nil {
		t.Fatal("expected collision error across button.css and modal.css, got nil")
	}

	msg := checkErr.Error()
	for _, want := range []string{
		`duplicate CSS variable "--brand"`,
		`duplicate CSS selector ".title"`,
		`duplicate CSS keyframes "pulse"`,
		"button.css:",
		"modal.css:",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("collision error missing %q:\n%s", want, msg)
		}
	}
}
