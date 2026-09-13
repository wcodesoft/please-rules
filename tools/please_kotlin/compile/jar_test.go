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
