package download

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractGzTar_ValidAndMaliciousPaths(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	entries := []struct {
		name    string
		content string
		isDir   bool
	}{
		{name: "good/dir/", isDir: true},
		{name: "good/dir/file.txt", content: "hello world"},
		{name: "/absolute/path.txt", content: "evil absolute"},
		{name: "../traversal.txt", content: "evil traversal"},
	}

	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     0644,
			Typeflag: tar.TypeReg,
		}
		if e.isDir {
			hdr.Typeflag = tar.TypeDir
			hdr.Mode = 0755
		} else {
			hdr.Size = int64(len(e.content))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if !e.isDir {
			if _, err := tw.Write([]byte(e.content)); err != nil {
				t.Fatal(err)
			}
		}
	}
	tw.Close()
	gw.Close()

	destDir := t.TempDir()
	if err := extractGzTar(bytes.NewReader(buf.Bytes()), destDir); err != nil {
		t.Fatalf("extractGzTar failed: %v", err)
	}

	goodPath := filepath.Join(destDir, "good", "dir", "file.txt")
	data, err := os.ReadFile(goodPath)
	if err != nil {
		t.Fatalf("reading extracted file: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("got %q, want %q", string(data), "hello world")
	}

	// Malicious paths must not exist
	if _, err := os.Stat(filepath.Join(destDir, "absolute")); err == nil {
		t.Errorf("absolute path was not skipped")
	}
}

func TestExtractGzTar_InvalidGzip(t *testing.T) {
	destDir := t.TempDir()
	err := extractGzTar(bytes.NewReader([]byte("not a gzip")), destDir)
	if err == nil {
		t.Error("expected error for invalid gzip stream, got nil")
	}
}

func TestExtractTarEntry_Symlink(t *testing.T) {
	destDir := t.TempDir()
	hdr := &tar.Header{
		Name:     "link_to_nowhere",
		Typeflag: tar.TypeSymlink,
		Linkname: "/etc/passwd",
	}
	err := extractTarEntry(nil, hdr, destDir)
	if err != nil {
		t.Fatalf("unexpected error for symlink: %v", err)
	}
}

func TestPackTar_RoundTrip(t *testing.T) {
	baseDir := t.TempDir()
	subDir := "mycrate-1.0.0"
	crateDir := filepath.Join(baseDir, subDir)
	if err := os.MkdirAll(filepath.Join(crateDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crateDir, "src", "lib.rs"), []byte("fn foo() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	outTar := filepath.Join(baseDir, "output.tar")
	if err := packTar(baseDir, subDir, outTar); err != nil {
		t.Fatalf("packTar failed: %v", err)
	}

	f, err := os.Open(outTar)
	if err != nil {
		t.Fatalf("opening outTar: %v", err)
	}
	defer f.Close()

	tr := tar.NewReader(f)
	foundLib := false
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		if hdr.Name == subDir+"/src/lib.rs" {
			foundLib = true
		}
	}
	if !foundLib {
		t.Errorf("expected %s/src/lib.rs in tar archive", subDir)
	}
}

func TestPackTar_InvalidOutPath(t *testing.T) {
	baseDir := t.TempDir()
	err := packTar(baseDir, "sub", "/dev/null/cannot/create/file.tar")
	if err == nil {
		t.Fatal("expected error for invalid outPath, got nil")
	}
}
