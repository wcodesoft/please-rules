package unpack

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestUnpackTarball(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpack_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tarPath := filepath.Join(tmpDir, "pkg.tgz")
	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	pkgJSON := `{"name": "test-pkg", "main": "index.js", "types": "index.d.ts"}`
	hdr := &tar.Header{
		Name: "package/package.json",
		Mode: 0644,
		Size: int64(len(pkgJSON)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(pkgJSON)); err != nil {
		t.Fatal(err)
	}

	jsContent := `console.log("hello");`
	hdr2 := &tar.Header{
		Name: "package/index.js",
		Mode: 0644,
		Size: int64(len(jsContent)),
	}
	if err := tw.WriteHeader(hdr2); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(jsContent)); err != nil {
		t.Fatal(err)
	}

	tw.Close()
	gzw.Close()
	f.Close()

	outDir := filepath.Join(tmpDir, "out")
	opts := Options{
		Tarball: tarPath,
		Out:     outDir,
		Name:    "test-pkg",
	}

	if err := Run(opts); err != nil {
		t.Fatalf("unpack.Run failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "index.js")); err != nil {
		t.Errorf("expected extracted index.js: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "ts_module.json")); err != nil {
		t.Errorf("expected generated ts_module.json: %v", err)
	}
}
