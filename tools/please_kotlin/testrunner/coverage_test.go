package testrunner

import (
	"strings"
	"testing"
)

func TestParseJacocoXml(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<report name="test">
  <package name="org/example">
    <sourcefile name="Math.kt">
      <line nr="10" mi="0" ci="1" mb="0" cb="0"/>
      <line nr="15" mi="1" ci="0" mb="0" cb="0"/>
    </sourcefile>
  </package>
</report>`

	covs, err := ParseJacocoXml([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParseJacocoXml failed: %v", err)
	}

	if len(covs) != 1 {
		t.Fatalf("expected 1 file, got %d", len(covs))
	}
	f := covs[0]
	if !strings.HasSuffix(f.Path, "Math.kt") {
		t.Errorf("expected path ending with Math.kt, got %s", f.Path)
	}
	if f.LineHits[10] != 1 {
		t.Errorf("expected line 10 hits = 1, got %d", f.LineHits[10])
	}
	if f.LineHits[15] != 0 {
		t.Errorf("expected line 15 hits = 0, got %d", f.LineHits[15])
	}
}

func TestNormalizeCoveragePaths(t *testing.T) {
	covs := []FileCoverage{
		{
			Path:     "org/example/Math.kt",
			LineHits: map[int]int{1: 1},
		},
	}
	known := []string{"src/org/example/Math.kt"}
	normalized := NormalizeCoveragePaths(covs, "", known)

	if len(normalized) != 1 {
		t.Fatalf("expected 1, got %d", len(normalized))
	}
	if normalized[0].Path != "src/org/example/Math.kt" {
		t.Errorf("got %q, want src/org/example/Math.kt", normalized[0].Path)
	}
}

func TestFormatGcov(t *testing.T) {
	covs := []FileCoverage{
		{
			Path:     "test.kt",
			LineHits: map[int]int{1: 2, 2: 0},
		},
	}
	gcov := FormatGcov(covs, "")
	str := string(gcov)
	if !strings.Contains(str, "Source:test.kt") {
		t.Errorf("missing Source:test.kt in %s", str)
	}
	if !strings.Contains(str, "2:    1:code") {
		t.Errorf("missing hit line in %s", str)
	}
	if !strings.Contains(str, "#####:    2:code") {
		t.Errorf("missing unhit line in %s", str)
	}
}

func TestFormatLcov(t *testing.T) {
	covs := []FileCoverage{
		{
			Path:     "test.kt",
			LineHits: map[int]int{1: 2, 2: 0},
		},
	}
	lcov := FormatLcov(covs)
	str := string(lcov)
	if !strings.Contains(str, "SF:test.kt") {
		t.Errorf("missing SF:test.kt in %s", str)
	}
	if !strings.Contains(str, "DA:1,2") {
		t.Errorf("missing DA:1,2 in %s", str)
	}
	if !strings.Contains(str, "DA:2,0") {
		t.Errorf("missing DA:2,0 in %s", str)
	}
}
