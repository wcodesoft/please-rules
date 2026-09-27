package unpack

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/please_ts/importmap"
)

func createTestTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		tarName := name
		if !strings.HasPrefix(tarName, "package/") {
			tarName = "package/" + tarName
		}
		hdr := &tar.Header{
			Name: tarName,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	_ = tw.Close()
	_ = gzw.Close()
	return buf.Bytes()
}

func TestUnpackTarball(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpack_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tarPath := filepath.Join(tmpDir, "pkg.tgz")
	tarBytes := createTestTarball(t, map[string]string{
		"package.json": `{"name": "test-pkg", "main": "index.js", "types": "index.d.ts"}`,
		"index.js":     `console.log("hello");`,
	})
	if err := os.WriteFile(tarPath, tarBytes, 0644); err != nil {
		t.Fatal(err)
	}

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

func TestUnpackWithDiscoveredLocalDeps(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpack_localdeps_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tarPath := filepath.Join(tmpDir, "pkg.tgz")
	tarBytes := createTestTarball(t, map[string]string{
		"package.json": `{"name": "parent-pkg", "main": "index.js"}`,
		"index.js":     `require("child");`,
	})
	if err := os.WriteFile(tarPath, tarBytes, 0644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(tmpDir, "out")
	// Pre-create a local dependency in .deps/child
	childDir := filepath.Join(outDir, ".deps", "child")
	_ = os.MkdirAll(childDir, 0755)
	_ = os.WriteFile(filepath.Join(childDir, "package.json"), []byte(`{"name": "child", "main": "index.js"}`), 0644)
	_ = os.WriteFile(filepath.Join(childDir, "index.js"), []byte(`module.exports = 'child';`), 0644)

	opts := Options{
		Tarball: tarPath,
		Out:     outDir,
		Name:    "parent-pkg",
	}

	if err := Run(opts); err != nil {
		t.Fatalf("unpack.Run failed: %v", err)
	}

	metaBytes, err := os.ReadFile(filepath.Join(outDir, "ts_module.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta importmap.ModuleMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatal(err)
	}

	expectedDep := "./.deps/child/index.js"
	if got := meta.Deps["child"]; got != expectedDep {
		t.Errorf("expected meta.Deps['child'] = %q, got %q", expectedDep, got)
	}
}

func TestUnpackWithMockRegistry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpack_mockreg_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create helper-pkg tarball
	helperTarball := createTestTarball(t, map[string]string{
		"package.json": `{"name": "helper-pkg", "version": "1.0.0", "main": "index.js"}`,
		"index.js":     `module.exports = { helper: true };`,
	})

	// Start mock NPM registry
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/helper-pkg") {
			manifest := fmt.Sprintf(`{
				"name": "helper-pkg",
				"dist-tags": { "latest": "1.0.0" },
				"versions": {
					"1.0.0": {
						"version": "1.0.0",
						"dist": { "tarball": "%s/helper-pkg/-/helper-pkg-1.0.0.tgz" }
					}
				}
			}`, server.URL)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(manifest))
			return
		}
		if strings.HasSuffix(r.URL.Path, "helper-pkg-1.0.0.tgz") {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(helperTarball)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	// Root package that depends on helper-pkg
	rootTarball := createTestTarball(t, map[string]string{
		"package.json": `{"name": "root-pkg", "version": "2.0.0", "main": "index.js", "dependencies": {"helper-pkg": "^1.0.0"}}`,
		"index.js":     `const h = require("helper-pkg");`,
	})

	rootTarPath := filepath.Join(tmpDir, "root.tgz")
	if err := os.WriteFile(rootTarPath, rootTarball, 0644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(tmpDir, "out")
	opts := Options{
		Tarball:           rootTarPath,
		Out:               outDir,
		Name:              "root-pkg",
		ResolveTransitive: true,
		Registry:          server.URL,
	}

	if err := Run(opts); err != nil {
		t.Fatalf("unpack.Run failed with mock registry: %v", err)
	}

	// Verify helper-pkg was downloaded and unpacked into .deps/helper-pkg
	if _, err := os.Stat(filepath.Join(outDir, ".deps", "helper-pkg", "index.js")); err != nil {
		t.Errorf("expected downloaded .deps/helper-pkg/index.js: %v", err)
	}

	// Verify ts_module.json records the dependency
	metaBytes, err := os.ReadFile(filepath.Join(outDir, "ts_module.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta importmap.ModuleMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatal(err)
	}

	expectedDep := "./.deps/helper-pkg/index.js"
	if got := meta.Deps["helper-pkg"]; got != expectedDep {
		t.Errorf("expected meta.Deps['helper-pkg'] = %q, got %q", expectedDep, got)
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
