package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLcov(t *testing.T) {
	rawLcov := `TN:
SF:/path/to/repo/src/lib.rs
DA:1,2
DA:2,0
DA:5,1
end_of_record
SF:/rustc/12345/library/core/src/lib.rs
DA:1,1
end_of_record
`
	files, err := ParseLcov([]byte(rawLcov))
	if err != nil {
		t.Fatalf("ParseLcov failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].Path != "/path/to/repo/src/lib.rs" {
		t.Errorf("unexpected path: %s", files[0].Path)
	}
	if len(files[0].LineHits) != 3 {
		t.Errorf("expected 3 line hits, got %d", len(files[0].LineHits))
	}
	if files[0].LineHits[1] != 2 || files[0].LineHits[2] != 0 || files[0].LineHits[5] != 1 {
		t.Errorf("unexpected line hits: %+v", files[0].LineHits)
	}

	norm := NormalizeLcovPaths(files, "/path/to/repo")
	if len(norm) != 1 {
		t.Fatalf("expected 1 file after normalization (rustc filtered), got %d", len(norm))
	}
	if norm[0].Path != "src/lib.rs" {
		t.Errorf("expected normalized path 'src/lib.rs', got '%s'", norm[0].Path)
	}

	gcov := FormatGcov(norm, "")
	gcovStr := string(gcov)
	if !strings.Contains(gcovStr, "Source:src/lib.rs") {
		t.Errorf("expected Source:src/lib.rs in gcov output, got:\n%s", gcovStr)
	}
	if !strings.Contains(gcovStr, "2:    1:code") && !strings.Contains(gcovStr, "2:  1:code") {
		t.Errorf("expected line 1 covered with 2 hits, got:\n%s", gcovStr)
	}
	if !strings.Contains(gcovStr, "#####:    2:code") && !strings.Contains(gcovStr, "#####:  2:code") {
		t.Errorf("expected line 2 uncovered (#####), got:\n%s", gcovStr)
	}
	if !strings.Contains(gcovStr, "-:    3:code") && !strings.Contains(gcovStr, "-:  3:code") {
		t.Errorf("expected line 3 not executable (-), got:\n%s", gcovStr)
	}
}

func TestProcessCoverage(t *testing.T) {
	rawLcov := `SF:/repo/root/test.rs
DA:1,1
end_of_record
`
	normLcov, gcov, err := ProcessCoverage([]byte(rawLcov), "/repo/root")
	if err != nil {
		t.Fatalf("ProcessCoverage failed: %v", err)
	}
	if !strings.Contains(string(normLcov), "SF:test.rs\nDA:1,1\nend_of_record") {
		t.Errorf("unexpected normLcov: %s", string(normLcov))
	}
	if !strings.HasPrefix(string(gcov), "        -:    0:Source:test.rs\n") {
		t.Errorf("unexpected gcov output: %s", string(gcov))
	}
}

func TestNormalizePathStrategies(t *testing.T) {
	// External paths
	if got := normalizePath("/rustc/abc/src/lib.rs", ""); got != "" {
		t.Errorf("expected empty for rustc path, got %s", got)
	}
	if got := normalizePath("/home/user/.cargo/registry/src/lib.rs", ""); got != "" {
		t.Errorf("expected empty for cargo path, got %s", got)
	}
	if got := normalizePath("/usr/lib/rustlib/src/lib.rs", ""); got != "" {
		t.Errorf("expected empty for rustlib path, got %s", got)
	}

	// Please build dir
	if got := normalizePath("/tmp/plz/._build/tools/test.rs", ""); got != "tools/test.rs" {
		t.Errorf("expected tools/test.rs, got %s", got)
	}
	if got := normalizePath("/tmp/plz/._test/run_1/tools/test.rs", ""); got != "tools/test.rs" {
		t.Errorf("expected tools/test.rs, got %s", got)
	}

	// Repo relative
	mockRepo := "/repo/root"
	sourceFile := filepath.Join(mockRepo, "src", "main.rs")
	if got := normalizePath(sourceFile, mockRepo); got != "src/main.rs" {
		t.Errorf("expected src/main.rs, got %s", got)
	}

	// Relative path
	if got := normalizePath("src/lib.rs", ""); got != "src/lib.rs" {
		t.Errorf("expected src/lib.rs, got %s", got)
	}
	if got := normalizePath("./src/lib.rs", ""); got != "src/lib.rs" {
		t.Errorf("expected src/lib.rs, got %s", got)
	}

	// Candidate matching inside repo
	tmpRepo := t.TempDir()
	subDir := filepath.Join(tmpRepo, "nested", "pkg")
	_ = os.MkdirAll(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "mod.rs"), []byte("fn foo() {}"), 0644)
	fakeSandboxPath := "/sandbox/tmp/plz-out/tmp/nested/pkg/mod.rs"
	if got := normalizePath(fakeSandboxPath, tmpRepo); got != "nested/pkg/mod.rs" {
		t.Errorf("expected nested/pkg/mod.rs candidate match, got %s", got)
	}
}
