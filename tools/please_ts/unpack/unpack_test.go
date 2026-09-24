package unpack

import (
	"archive/tar"
	"archive/zip"
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

func TestUnpackZipToolchain(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpack_toolchain_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "toolchain.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	zw := zip.NewWriter(f)
	binHeader := &zip.FileHeader{
		Name: "chrome-headless-shell-linux64/chrome-headless-shell",
	}
	w, err := zw.CreateHeader(binHeader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("#!/bin/sh\necho ok\n")); err != nil {
		t.Fatal(err)
	}

	libHeader := &zip.FileHeader{
		Name: "chrome-headless-shell-linux64/libEGL.so",
	}
	w2, err := zw.CreateHeader(libHeader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w2.Write([]byte("fake-lib")); err != nil {
		t.Fatal(err)
	}

	zw.Close()
	f.Close()

	outDir := filepath.Join(tmpDir, "bin")
	opts := Options{
		Archive: zipPath,
		Out:     outDir,
		Binary:  "chrome-headless-shell",
		Symlink: "chromium",
	}

	if err := Run(opts); err != nil {
		t.Fatalf("unpack.Run toolchain failed: %v", err)
	}

	binStat, err := os.Stat(filepath.Join(outDir, "chrome-headless-shell"))
	if err != nil {
		t.Fatalf("expected extracted binary: %v", err)
	}
	if binStat.Mode().Perm()&0111 == 0 {
		t.Errorf("expected binary to be executable, got mode: %v", binStat.Mode())
	}

	if _, err := os.Stat(filepath.Join(outDir, "libEGL.so")); err != nil {
		t.Errorf("expected sibling library libEGL.so to be copied: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "chromium")); err != nil {
		t.Errorf("expected chromium symlink: %v", err)
	}
}
