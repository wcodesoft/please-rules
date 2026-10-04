package unpack

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"encoding/base64"
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

func TestCleanRelativePathMatrix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"foo/bar", "./foo/bar"},
		{"./foo/bar", "./foo/bar"},
		{"../foo/bar", "../foo/bar"},
		{"/abs/path", "/abs/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanRelativePath(tt.input)
			if got != tt.want {
				t.Errorf("cleanRelativePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnpackArchiveMatrix(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		pkgName   string
		wantEntry string
		wantTypes string
	}{
		{
			name: "package with main and types",
			files: map[string]string{
				"package.json": `{"name": "test-pkg", "main": "index.js", "types": "index.d.ts"}`,
				"index.js":     `console.log("hello");`,
				"index.d.ts":   `export declare const hello: string;`,
			},
			pkgName:   "test-pkg",
			wantEntry: "index.js",
			wantTypes: "index.d.ts",
		},
		{
			name: "package with module field and typings alias",
			files: map[string]string{
				"package.json":    `{"name": "esm-pkg", "module": "dist/index.mjs", "typings": "dist/index.d.ts"}`,
				"dist/index.mjs":  `export const foo = 1;`,
				"dist/index.d.ts": `export declare const foo: number;`,
			},
			pkgName:   "esm-pkg",
			wantEntry: "dist/index.mjs",
			wantTypes: "dist/index.d.ts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "unpack_archive_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			tarPath := filepath.Join(tmpDir, "pkg.tgz")
			tarBytes := createTestTarball(t, tt.files)
			if err := os.WriteFile(tarPath, tarBytes, 0644); err != nil {
				t.Fatal(err)
			}

			outDir := filepath.Join(tmpDir, "out")
			opts := Options{
				Tarball: tarPath,
				Out:     outDir,
				Name:    tt.pkgName,
			}

			if err := Run(opts); err != nil {
				t.Fatalf("unpack.Run failed: %v", err)
			}

			metaBytes, err := os.ReadFile(filepath.Join(outDir, "ts_module.json"))
			if err != nil {
				t.Fatalf("failed reading ts_module.json: %v", err)
			}
			var meta importmap.ModuleMetadata
			if err := json.Unmarshal(metaBytes, &meta); err != nil {
				t.Fatalf("failed unmarshaling ts_module.json: %v", err)
			}

			if meta.Name != tt.pkgName {
				t.Errorf("meta.Name = %q, want %q", meta.Name, tt.pkgName)
			}
			if meta.Entry != tt.wantEntry {
				t.Errorf("meta.Entry = %q, want %q", meta.Entry, tt.wantEntry)
			}
			if meta.Types != tt.wantTypes {
				t.Errorf("meta.Types = %q, want %q", meta.Types, tt.wantTypes)
			}
		})
	}
}

func TestUnpackWithDiscoveredLocalDepsMatrix(t *testing.T) {
	tests := []struct {
		name      string
		localDeps map[string]map[string]string // pkgName -> relFile -> content
		wantDeps  map[string]string
	}{
		{
			name: "unscoped and scoped local dependencies",
			localDeps: map[string]map[string]string{
				"child": {
					"package.json": `{"name": "child", "main": "index.js"}`,
					"index.js":     `module.exports = 'child';`,
				},
				"@scope/helper": {
					"package.json": `{"name": "@scope/helper", "main": "index.js"}`,
					"index.js":     `module.exports = 'scoped';`,
				},
			},
			wantDeps: map[string]string{
				"child":         "./.deps/child/index.js",
				"@scope/helper": "./.deps/@scope/helper/index.js",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Pre-stage local dependencies in .deps/
			for depName, files := range tt.localDeps {
				depDir := filepath.Join(outDir, ".deps", depName)
				_ = os.MkdirAll(depDir, 0755)
				for relPath, content := range files {
					filePath := filepath.Join(depDir, relPath)
					_ = os.MkdirAll(filepath.Dir(filePath), 0755)
					_ = os.WriteFile(filePath, []byte(content), 0644)
				}
			}

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

			for depName, want := range tt.wantDeps {
				got, ok := meta.Deps[depName]
				if !ok {
					t.Errorf("meta.Deps[%q] missing, want %q", depName, want)
				} else if got != want {
					t.Errorf("meta.Deps[%q] = %q, want %q", depName, got, want)
				}
			}
		})
	}
}

func TestUnpackWithMockRegistryMatrix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpack_mockreg_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	helperTarball := createTestTarball(t, map[string]string{
		"package.json": `{"name": "helper-pkg", "version": "1.0.0", "main": "index.js"}`,
		"index.js":     `module.exports = { helper: true };`,
	})

	helperSum := sha512.Sum512(helperTarball)
	helperIntegrity := "sha512-" + base64.StdEncoding.EncodeToString(helperSum[:])
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/helper-pkg") {
			manifest := fmt.Sprintf(`{
				"name": "helper-pkg",
				"dist-tags": { "latest": "1.0.0" },
				"versions": {
					"1.0.0": {
						"version": "1.0.0",
						"dist": { "tarball": "%s/helper-pkg/-/helper-pkg-1.0.0.tgz", "integrity": "%s" }
					}
				}
			}`, server.URL, helperIntegrity)
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

	metaBytes, err := os.ReadFile(filepath.Join(outDir, "ts_module.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta importmap.ModuleMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatal(err)
	}

	wantDep := "./.deps/helper-pkg/index.js"
	if got := meta.Deps["helper-pkg"]; got != wantDep {
		t.Errorf("meta.Deps['helper-pkg'] = %q, want %q", got, wantDep)
	}
}

func TestUnpackToolchainMatrix(t *testing.T) {
	tests := []struct {
		name        string
		binaryName  string
		symlinkName string
		wantBinMode os.FileMode
	}{
		{
			name:        "extracts binary with executable perm and creates symlink",
			binaryName:  "chrome-headless-shell",
			symlinkName: "chromium",
			wantBinMode: 0755,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "unpack_toolchain_matrix_*")
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
				Binary:  tt.binaryName,
				Symlink: tt.symlinkName,
			}

			if err := Run(opts); err != nil {
				t.Fatalf("unpack.Run toolchain failed: %v", err)
			}

			binStat, err := os.Stat(filepath.Join(outDir, tt.binaryName))
			if err != nil {
				t.Fatalf("expected extracted binary: %v", err)
			}
			if binStat.Mode().Perm()&0111 == 0 {
				t.Errorf("binary mode = %v, want executable", binStat.Mode())
			}

			if _, err := os.Stat(filepath.Join(outDir, "libEGL.so")); err != nil {
				t.Errorf("expected sibling library libEGL.so to be copied: %v", err)
			}

			if tt.symlinkName != "" {
				if _, err := os.Stat(filepath.Join(outDir, tt.symlinkName)); err != nil {
					t.Errorf("expected symlink %q to exist: %v", tt.symlinkName, err)
				}
			}
		})
	}
}
