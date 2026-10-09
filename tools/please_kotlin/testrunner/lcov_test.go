package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/common/lcov"
)

// Trimmed from a JaCoCo run over a small fixture: an if/else-if chain (classify), a
// short-circuit expression (both) and a function that never ran (unused). The tests
// called classify(-1), classify(5) and both(false, true).
const jacocoSample = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<report name="JaCoCo Coverage Report">
  <package name="pkg">
    <class name="pkg/LibKt" sourcefilename="Lib.kt">
      <method name="classify" desc="(I)Ljava/lang/String;" line="2">
        <counter type="BRANCH" missed="1" covered="3"/>
        <counter type="METHOD" missed="0" covered="1"/>
      </method>
      <method name="both" desc="(ZZ)Z" line="12">
        <counter type="BRANCH" missed="3" covered="1"/>
        <counter type="METHOD" missed="0" covered="1"/>
      </method>
      <method name="unused" desc="()I" line="16">
        <counter type="METHOD" missed="1" covered="0"/>
      </method>
      <method name="&lt;init&gt;" desc="()V">
        <counter type="METHOD" missed="1" covered="0"/>
      </method>
    </class>
    <sourcefile name="Lib.kt">
      <line nr="2" mi="0" ci="3" mb="0" cb="2"/>
      <line nr="3" mi="0" ci="2" mb="0" cb="0"/>
      <line nr="4" mi="0" ci="2" mb="1" cb="1"/>
      <line nr="5" mi="2" ci="0" mb="0" cb="0"/>
      <line nr="12" mi="4" ci="4" mb="3" cb="1"/>
      <line nr="16" mi="2" ci="0" mb="0" cb="0"/>
      <line nr="17" mi="0" ci="0" mb="0" cb="0"/>
    </sourcefile>
  </package>
</report>`

func TestJacocoToLcov(t *testing.T) {
	r, err := JacocoToLcov([]byte(jacocoSample), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	f := r.File("pkg/Lib.kt")
	if f == nil {
		t.Fatalf("pkg/Lib.kt missing, got %v", r.Files)
	}

	wantFns := map[string]lcov.Function{
		"pkg.LibKt.classify(I)Ljava/lang/String;": {Name: "pkg.LibKt.classify(I)Ljava/lang/String;", Line: 2, Hits: 1},
		"pkg.LibKt.both(ZZ)Z":                     {Name: "pkg.LibKt.both(ZZ)Z", Line: 12, Hits: 1},
		"pkg.LibKt.unused()I":                     {Name: "pkg.LibKt.unused()I", Line: 16, Hits: 0},
	}
	if len(f.Functions) != len(wantFns) {
		t.Fatalf("functions = %v, want %d (methods without a line are skipped)", f.Functions, len(wantFns))
	}
	for _, fn := range f.Functions {
		if want, ok := wantFns[fn.Name]; !ok || want != fn {
			t.Errorf("unexpected function %+v", fn)
		}
	}

	if _, ok := f.Lines[17]; ok {
		t.Error("a line with no instructions must not be recorded")
	}
	for line, want := range map[int]lcov.Status{
		2:  lcov.Covered,   // both arms ran
		3:  lcov.Covered,   // no branches
		4:  lcov.Partial,   // one arm missed
		5:  lcov.Uncovered, // never ran
		12: lcov.Partial,   // short-circuit: three of four arms missed
		16: lcov.Uncovered,
	} {
		if got := f.LineStatus(line); got != want {
			t.Errorf("LineStatus(%d) = %v, want %v", line, got, want)
		}
	}
	if len(f.Branches) != 2+2+4 || f.BranchesHit() != 2+1+1 {
		t.Errorf("branches = %d, hit %d; want 8 and 4", len(f.Branches), f.BranchesHit())
	}
}

func TestJacocoToLcovMarksBranchesOfUnexecutedLinesNotEvaluated(t *testing.T) {
	r, err := JacocoToLcov([]byte(`<report><package name=""><sourcefile name="a.kt">
		<line nr="3" mi="4" ci="0" mb="2" cb="0"/></sourcefile></package></report>`), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	f := r.File("a.kt")
	if f == nil || len(f.Branches) != 2 || f.Branches[0].Taken != lcov.NotEvaluated || f.LineStatus(3) != lcov.Uncovered {
		t.Errorf("unexpected report: %+v", f)
	}
}

func TestJacocoToLcovMapsPathsLikeTheGcovExport(t *testing.T) {
	r, err := JacocoToLcov([]byte(jacocoSample), "", []string{"src/main/Lib.kt"})
	if err != nil {
		t.Fatal(err)
	}
	if r.File("src/main/Lib.kt") == nil {
		t.Errorf("expected the known source path, got %v", r.Files[0].Path)
	}
}

func TestJacocoToLcovRejectsInvalidXML(t *testing.T) {
	if _, err := JacocoToLcov([]byte("<report>"), "", nil); err == nil {
		t.Error("expected an error")
	}
}

func TestWriteLcovFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "t.lcov")
	if err := writeLcovFile(out, []byte(jacocoSample), "", nil); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	for _, want := range []string{"SF:pkg/Lib.kt\n", "FNDA:0,pkg.LibKt.unused()I\n", "BRDA:4,0,0,1\n", "BRDA:4,0,1,0\n", "DA:5,0\n"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if err := writeLcovFile("", []byte("not xml"), "", nil); err != nil {
		t.Errorf("empty path must be a no-op, got %v", err)
	}
}
