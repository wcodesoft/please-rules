package compile

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateJar(t *testing.T) {
	tmpDir := t.TempDir()
	classesDir := filepath.Join(tmpDir, "classes")
	if err := os.MkdirAll(filepath.Join(classesDir, "pkg"), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	classFile := filepath.Join(classesDir, "pkg", "Test.class")
	if err := os.WriteFile(classFile, []byte("fake class bytecode"), 0644); err != nil {
		t.Fatalf("failed to write class file: %v", err)
	}

	outJar := filepath.Join(tmpDir, "out.jar")
	opts := JarOptions{
		SourceDir: classesDir,
		OutJar:    outJar,
		MainClass: "pkg.Test",
	}

	if err := CreateJar(opts); err != nil {
		t.Fatalf("CreateJar failed: %v", err)
	}

	zr, err := zip.OpenReader(outJar)
	if err != nil {
		t.Fatalf("failed to open generated jar: %v", err)
	}
	defer zr.Close()

	var foundManifest, foundClass bool
	for _, f := range zr.File {
		if f.Name == "META-INF/MANIFEST.MF" {
			foundManifest = true
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open manifest: %v", err)
			}
			content, _ := io.ReadAll(rc)
			rc.Close()
			if !strings.Contains(string(content), "Main-Class: pkg.Test") {
				t.Errorf("manifest missing Main-Class: %s", string(content))
			}
		}
		if f.Name == "pkg/Test.class" {
			foundClass = true
		}
	}

	if !foundManifest {
		t.Errorf("expected META-INF/MANIFEST.MF in jar")
	}
	if !foundClass {
		t.Errorf("expected pkg/Test.class in jar")
	}
}

func TestProjectClassesEntryListsOnlyProjectClasses(t *testing.T) {
	tmpDir := t.TempDir()
	classesDir := filepath.Join(tmpDir, "classes")
	if err := os.MkdirAll(filepath.Join(classesDir, "pkg"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(classesDir, "pkg", "Lib.class"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	libJar := filepath.Join(tmpDir, "lib.jar")
	if err := CreateJar(JarOptions{SourceDir: classesDir, OutJar: libJar, ExtraEntries: projectClassesEntry(classesDir, Options{Out: libJar})}); err != nil {
		t.Fatal(err)
	}
	if got := ReadProjectClasses(libJar); len(got) != 1 || got[0] != "pkg/Lib.class" {
		t.Fatalf("library jar project classes = %v, want [pkg/Lib.class]", got)
	}

	// A test bundle inherits its dependencies' classes but not its own.
	testClasses := filepath.Join(tmpDir, "test-classes")
	if err := os.MkdirAll(testClasses, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testClasses, "LibTest.class"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	testJar := filepath.Join(tmpDir, "test.jar")
	entries := projectClassesEntry(testClasses, Options{Out: testJar, Deps: []string{libJar}, MergeDeps: true})
	if err := CreateJar(JarOptions{SourceDir: testClasses, OutJar: testJar, MergeJars: []string{libJar}, ExtraEntries: entries}); err != nil {
		t.Fatal(err)
	}
	if got := ReadProjectClasses(testJar); len(got) != 1 || got[0] != "pkg/Lib.class" {
		t.Fatalf("test jar project classes = %v, want [pkg/Lib.class]", got)
	}
}

func TestReadProjectClassesWithoutEntry(t *testing.T) {
	jar := filepath.Join(t.TempDir(), "plain.jar")
	if err := CreateJar(JarOptions{SourceDir: t.TempDir(), OutJar: jar}); err != nil {
		t.Fatal(err)
	}
	if got := ReadProjectClasses(jar); got != nil {
		t.Errorf("ReadProjectClasses = %v, want nil", got)
	}
}
