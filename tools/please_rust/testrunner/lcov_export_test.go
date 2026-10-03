package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/common/lcov"
)

// llvm-cov lcov export (stable rustc, no branch regions) for a small fixture:
// classify(-1), classify(5) and both(false, true) ran; unused never did. The file is
// in the build sandbox, and a toolchain file is included too.
const rustLcovExport = `SF:/work/plz-out/tmp/pkg/lib._build/pkg/lib.rs
FN:11,both
FN:1,classify
FN:15,unused
FNDA:1,both
FNDA:2,classify
FNDA:0,unused
FNF:3
FNH:2
DA:1,2
DA:2,2
DA:3,1
DA:4,1
DA:5,0
DA:7,1
DA:11,1
DA:12,1
DA:15,0
LF:9
LH:7
BRF:0
BRH:0
end_of_record
SF:/rustc/abc123/library/core/src/fmt/mod.rs
FN:1,fmt
FNDA:3,fmt
DA:1,3
end_of_record
`

// The matching JSON export: line 12 is `    a && b`, whose right side never ran.
const rustJSONExport = `{"data":[{"files":[{"filename":"/work/plz-out/tmp/pkg/lib._build/pkg/lib.rs","segments":[
[1,1,2,true,true,false],[2,13,0,false,false,false],[3,9,1,true,true,false],[3,14,0,false,false,false],
[4,15,1,true,true,false],[4,21,0,false,false,false],[5,9,0,true,true,false],[5,15,0,false,false,false],
[7,9,1,true,true,false],[7,14,0,false,false,false],[9,1,2,true,true,false],[9,2,0,false,false,false],
[11,1,1,true,true,false],[12,6,0,false,false,false],[12,10,0,true,true,false],[12,11,0,false,false,false],
[13,1,1,true,true,false],[13,2,0,false,false,false],[15,1,0,true,true,false],[17,2,0,false,false,false]]}]}]}`

func TestBuildRawLcov(t *testing.T) {
	data, err := buildRawLcov([]byte(rustLcovExport), []byte(rustJSONExport), "")
	if err != nil {
		t.Fatal(err)
	}
	report, err := lcov.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Files) != 1 {
		t.Fatalf("files = %d, want the toolchain file dropped", len(report.Files))
	}
	f := report.File("pkg/lib.rs")
	if f == nil {
		t.Fatalf("pkg/lib.rs missing, got %q", report.Files[0].Path)
	}
	if len(f.Functions) != 3 || f.FunctionsHit() != 2 {
		t.Errorf("functions = %v", f.Functions)
	}
	for line, want := range map[int]lcov.Status{
		12: lcov.Partial,   // right side of && never ran
		5:  lcov.Uncovered, // body of the untaken arm
		15: lcov.Uncovered, // function that never ran
		2:  lcov.Covered,
	} {
		if got := f.LineStatus(line); got != want {
			t.Errorf("LineStatus(%d) = %v, want %v", line, got, want)
		}
	}
}

func TestBuildRawLcovWithoutRegionData(t *testing.T) {
	data, err := buildRawLcov([]byte(rustLcovExport), nil, "")
	if err != nil {
		t.Fatal(err)
	}
	report, _ := lcov.Parse(strings.NewReader(string(data)))
	f := report.File("pkg/lib.rs")
	if f == nil || len(f.Branches) != 0 || f.LineStatus(12) != lcov.Covered {
		t.Errorf("without the JSON export the report must still be written, lines only: %+v", f)
	}
}

func TestBuildRawLcovInvalidInput(t *testing.T) {
	if _, err := buildRawLcov([]byte("DA:1,1\n"), nil, ""); err == nil {
		t.Error("expected an error for a record outside an SF section")
	}
	if _, err := buildRawLcov([]byte(rustLcovExport), []byte("not json"), ""); err == nil {
		t.Error("expected an error for an invalid JSON export")
	}
}

func TestWriteRawLcov(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "t.lcov")
	if err := writeRawLcov(path, []byte("SF:a\n")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "SF:a\n" {
		t.Errorf("got %q", got)
	}
	if err := writeRawLcov("", []byte("x")); err != nil {
		t.Errorf("an empty path must be a no-op, got %v", err)
	}
	// Resetting to empty, as RunWithOptions does before the test runs.
	if err := writeRawLcov(path, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); len(got) != 0 {
		t.Errorf("expected an empty file, got %q", got)
	}
}
