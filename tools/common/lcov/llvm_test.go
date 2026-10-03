package lcov

import (
	"strings"
	"testing"
)

// Segments from `llvm-cov export -format=text` for a small Rust fixture (stable
// toolchain, no branch regions): classify(-1), classify(5) and both(false, true) ran;
// unused never did. Line 12 is `    a && b`: the region for `b` starts mid-line with a
// count of 0.
const rustExport = `{"data":[{"files":[{"filename":"/sandbox/lib.rs","segments":[
[1,1,2,true,true,false],[2,13,0,false,false,false],[3,9,1,true,true,false],[3,14,0,false,false,false],
[4,15,1,true,true,false],[4,21,0,false,false,false],[5,9,0,true,true,false],[5,15,0,false,false,false],
[7,9,1,true,true,false],[7,14,0,false,false,false],[9,1,2,true,true,false],[9,2,0,false,false,false],
[11,1,1,true,true,false],[12,6,0,false,false,false],[12,10,0,true,true,false],[12,11,0,false,false,false],
[13,1,1,true,true,false],[13,2,0,false,false,false],[15,1,0,true,true,false],[17,2,0,false,false,false]]}]}]}`

// The same fixture in Swift: line 4 is `    } else if n == 0 {` whose body never ran, and the
// right side of `a && b` on line 12 is a closure that never ran.
const swiftExport = `{"data":[{"files":[{"filename":"/sandbox/Lib.swift","segments":[
[1,42,2,true,true,false],[2,8,2,true,true,false],[2,13,2,true,false,false],[2,14,1,true,true,false],
[4,6,2,true,false,false],[4,15,1,true,true,false],[4,21,2,true,false,false],[4,22,0,true,true,false],
[6,6,2,true,false,false],[6,12,1,true,true,false],[8,6,2,true,false,false],[9,2,0,false,false,false],
[11,48,1,true,true,false],[12,17,0,true,true,false],[12,18,1,true,false,false],[13,2,0,false,false,false]]}]}]}`

func fileWithLines(path string, hits map[int]int) *Report {
	f := newFile(path)
	for l, h := range hits {
		f.Lines[l] = h
	}
	return &Report{Files: []*File{f}}
}

func norm(string) string { return "lib.rs" }

func TestAddRegionBranchesRust(t *testing.T) {
	r := fileWithLines("lib.rs", map[int]int{2: 2, 3: 1, 4: 1, 5: 0, 7: 1, 12: 1, 15: 0})
	if err := AddRegionBranches(r, strings.NewReader(rustExport), norm); err != nil {
		t.Fatal(err)
	}
	f := r.File("lib.rs")
	if len(f.Branches) != 1 || f.Branches[0] != (Branch{Line: 12, Block: 0, Arm: 0, Taken: 0}) {
		t.Fatalf("branches = %v, want only the never-run right side of line 12", f.Branches)
	}
	for line, want := range map[int]Status{12: Partial, 5: Uncovered, 4: Covered, 2: Covered} {
		if got := f.LineStatus(line); got != want {
			t.Errorf("LineStatus(%d) = %v, want %v", line, got, want)
		}
	}
}

func TestAddRegionBranchesSwift(t *testing.T) {
	r := fileWithLines("Lib.swift", map[int]int{1: 2, 2: 2, 3: 1, 4: 1, 5: 0, 6: 1, 7: 1, 8: 1, 12: 1})
	if err := AddRegionBranches(r, strings.NewReader(swiftExport), func(string) string { return "Lib.swift" }); err != nil {
		t.Fatal(err)
	}
	f := r.File("Lib.swift")
	want := []Branch{{4, 0, 0, 1}, {4, 0, 1, 0}, {12, 0, 0, 0}}
	if len(f.Branches) != len(want) {
		t.Fatalf("branches = %v, want %v", f.Branches, want)
	}
	for i, b := range want {
		if f.Branches[i] != b {
			t.Errorf("branch %d = %v, want %v", i, f.Branches[i], b)
		}
	}
	if f.LineStatus(4) != Partial || f.LineStatus(12) != Partial || f.LineStatus(2) != Covered || f.LineStatus(6) != Covered {
		t.Errorf("statuses: 4=%v 12=%v 2=%v 6=%v", f.LineStatus(4), f.LineStatus(12), f.LineStatus(2), f.LineStatus(6))
	}
}

func TestAddRegionBranchesKeepsExistingBranchesAndSkipsUnrunLines(t *testing.T) {
	r := fileWithLines("lib.rs", map[int]int{12: 1})
	r.Files[0].Branches = []Branch{{Line: 12, Block: 3, Arm: 0, Taken: 1}}
	if err := AddRegionBranches(r, strings.NewReader(rustExport), norm); err != nil {
		t.Fatal(err)
	}
	if got := r.Files[0].Branches; len(got) != 1 || got[0].Block != 3 {
		t.Errorf("existing branch data must be kept as is, got %v", got)
	}

	r = fileWithLines("lib.rs", map[int]int{12: 0})
	if err := AddRegionBranches(r, strings.NewReader(rustExport), norm); err != nil {
		t.Fatal(err)
	}
	if len(r.Files[0].Branches) != 0 {
		t.Errorf("a line that never ran is uncovered, not partial: %v", r.Files[0].Branches)
	}
}

func TestAddRegionBranchesIgnoresUnknownFilesAndGaps(t *testing.T) {
	r := fileWithLines("other.rs", map[int]int{1: 1})
	if err := AddRegionBranches(r, strings.NewReader(rustExport), norm); err != nil {
		t.Fatal(err)
	}
	if len(r.Files[0].Branches) != 0 {
		t.Errorf("file not in the export must be untouched: %v", r.Files[0].Branches)
	}

	gap := `{"data":[{"files":[{"filename":"a","segments":[[1,1,1,true,true,false],[2,5,0,true,true,true],[3,1,0,false,false,false]]}]}]}`
	r = fileWithLines("a", map[int]int{2: 1})
	if err := AddRegionBranches(r, strings.NewReader(gap), nil); err != nil {
		t.Fatal(err)
	}
	if len(r.Files[0].Branches) != 0 {
		t.Errorf("gap regions are not branches: %v", r.Files[0].Branches)
	}
}

func TestAddRegionBranchesInvalidInput(t *testing.T) {
	r := fileWithLines("a", map[int]int{1: 1})
	if err := AddRegionBranches(r, strings.NewReader("not json"), nil); err == nil {
		t.Error("expected an error for invalid JSON")
	}
	short := `{"data":[{"files":[{"filename":"a","segments":[[1,1,1]]}]}]}`
	if err := AddRegionBranches(r, strings.NewReader(short), nil); err == nil {
		t.Error("expected an error for a short segment")
	}
}
