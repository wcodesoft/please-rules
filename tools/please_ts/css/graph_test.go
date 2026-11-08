package css

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompileAndDiscoverGraphFilesOrdersDependenciesAndDeduplicatesDiamonds(t *testing.T) {
	root := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	// Create source files
	tokensSrc := filepath.Join(root, "src", "tokens.css")
	btnSrc := filepath.Join(root, "src", "btn.css")
	cardSrc := filepath.Join(root, "src", "card.css")
	pageSrc := filepath.Join(root, "src", "page.css")
	if err := os.MkdirAll(filepath.Dir(tokensSrc), 0755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(tokensSrc, []byte(":root { --gap: 8px; }"), 0644)
	_ = os.WriteFile(btnSrc, []byte(".btn { gap: var(--gap); }"), 0644)
	_ = os.WriteFile(cardSrc, []byte(".card { gap: var(--gap); }"), 0644)
	_ = os.WriteFile(pageSrc, []byte(".page { display: grid; }"), 0644)

	// Compile tokens css_library
	tokensOut := filepath.Join("ui", "tokens")
	if err := Compile(CompileOptions{
		Name: "ui/tokens:tokens",
		Out:  tokensOut,
		Srcs: []string{tokensSrc},
	}); err != nil {
		t.Fatalf("Compile tokens failed: %v", err)
	}

	// Compile btn css_library depending on tokens
	btnOut := filepath.Join("ui", "btn")
	if err := Compile(CompileOptions{
		Name: "ui/btn:btn",
		Out:  btnOut,
		Srcs: []string{btnSrc},
		Deps: []string{tokensOut},
	}); err != nil {
		t.Fatalf("Compile btn failed: %v", err)
	}

	// Compile card css_library depending on btn and tokens (diamond)
	cardOut := filepath.Join("ui", "card")
	if err := Compile(CompileOptions{
		Name: "ui/card:card",
		Out:  cardOut,
		Srcs: []string{cardSrc},
		Deps: []string{btnOut, tokensOut},
	}); err != nil {
		t.Fatalf("Compile card failed: %v", err)
	}

	got, err := DiscoverGraphFiles([]string{btnOut, cardOut}, []string{pageSrc}, ".")
	if err != nil {
		t.Fatalf("DiscoverGraphFiles failed: %v", err)
	}

	want := []string{
		filepath.Join(tokensOut, "tokens.css"),
		filepath.Join(btnOut, "btn.css"),
		filepath.Join(cardOut, "card.css"),
		filepath.Clean(pageSrc),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DiscoverGraphFiles =\n  %v\nwant:\n  %v", got, want)
	}
}

func TestCompileRejectsIntraLibraryCollisions(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.css")
	b := filepath.Join(dir, "b.css")
	_ = os.WriteFile(a, []byte(".dup { color: red; }"), 0644)
	_ = os.WriteFile(b, []byte(".dup { color: blue; }"), 0644)

	err := Compile(CompileOptions{
		Name: "ui:dup",
		Out:  filepath.Join(dir, "out"),
		Srcs: []string{a, b},
	})
	if err == nil || !strings.Contains(err.Error(), `duplicate CSS selector ".dup"`) {
		t.Fatalf("expected duplicate CSS selector error, got: %v", err)
	}
}
