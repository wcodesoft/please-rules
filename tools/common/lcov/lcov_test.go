package lcov

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// Captured from a V8 coverage run over a small fixture: an if/else-if chain, a
// short-circuit expression and a function that is never called.
const v8Sample = `TN:
SF:lib.ts
FN:1,classify
FN:11,both
FN:15,unused
FNF:3
FNH:2
FNDA:2,classify
FNDA:1,both
FNDA:0,unused
DA:2,2
DA:3,1
DA:4,1
DA:5,0
DA:7,1
DA:12,1
DA:16,0
LF:7
LH:5
BRDA:2,0,0,1
BRDA:2,0,1,1
BRDA:4,1,0,0
BRDA:4,1,1,1
BRDA:12,2,0,1
BRDA:12,2,1,0
BRF:6
BRH:4
end_of_record
`

func parse(t *testing.T, s string) *Report {
	t.Helper()
	r, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return r
}

func TestParseCapturedReport(t *testing.T) {
	r := parse(t, v8Sample)
	f := r.File("lib.ts")
	if f == nil {
		t.Fatal("lib.ts not found")
	}
	wantFns := []Function{{"classify", 1, 2}, {"both", 11, 1}, {"unused", 15, 0}}
	if !reflect.DeepEqual(f.Functions, wantFns) {
		t.Errorf("functions = %v, want %v", f.Functions, wantFns)
	}
	if f.LinesFound() != 7 || f.LinesHit() != 5 {
		t.Errorf("lines found/hit = %d/%d, want 7/5", f.LinesFound(), f.LinesHit())
	}
	if len(f.Branches) != 6 || f.BranchesHit() != 4 {
		t.Errorf("branches = %d, hit %d, want 6 and 4", len(f.Branches), f.BranchesHit())
	}
}

func TestLineStatus(t *testing.T) {
	f := parse(t, v8Sample).File("lib.ts")
	for line, want := range map[int]Status{
		1:  NotExecutable, // no record
		2:  Covered,       // both arms taken
		3:  Covered,
		4:  Partial, // ran, but one arm never did
		5:  Uncovered,
		12: Partial, // short-circuit: right side never evaluated
		16: Uncovered,
	} {
		if got := f.LineStatus(line); got != want {
			t.Errorf("LineStatus(%d) = %v, want %v", line, got, want)
		}
	}
}

func TestRoundTripIsStable(t *testing.T) {
	var first, second bytes.Buffer
	if err := parse(t, v8Sample).Write(&first); err != nil {
		t.Fatal(err)
	}
	if err := parse(t, first.String()).Write(&second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Errorf("write is not stable:\n%s\n--\n%s", first.String(), second.String())
	}
	for _, want := range []string{"FNF:3\n", "FNH:2\n", "BRF:6\n", "BRH:4\n", "LF:7\n", "LH:5\n", "BRDA:4,1,0,0\n"} {
		if !strings.Contains(first.String(), want) {
			t.Errorf("output missing %q:\n%s", want, first.String())
		}
	}
}

func TestParseFunctionRecordForms(t *testing.T) {
	r := parse(t, `SF:a.rs
FN:3,5,start_end_form
FN:9,name,with,commas
FNDA:4,start_end_form
FNDA:0,name,with,commas
end_of_record
`)
	got := r.File("a.rs").Functions
	want := []Function{{"start_end_form", 3, 4}, {"name,with,commas", 9, 0}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("functions = %v, want %v", got, want)
	}
}

func TestBranchNotEvaluated(t *testing.T) {
	f := parse(t, "SF:a\nDA:1,1\nBRDA:1,0,0,-\nBRDA:1,0,1,2\nend_of_record\n").File("a")
	if f.Branches[0].Taken != NotEvaluated || f.Branches[1].Taken != 2 {
		t.Fatalf("branches = %v", f.Branches)
	}
	if f.LineStatus(1) != Partial {
		t.Errorf("a line with a never-evaluated arm should be partial")
	}
	var out bytes.Buffer
	_ = (&Report{Files: []*File{f}}).Write(&out)
	if !strings.Contains(out.String(), "BRDA:1,0,0,-\n") {
		t.Errorf("not-evaluated branch must round-trip as '-':\n%s", out.String())
	}
}

func TestParseMergesRepeatedFileRecords(t *testing.T) {
	r := parse(t, "SF:a\nDA:1,1\nend_of_record\nSF:a\nDA:1,2\nDA:2,0\nend_of_record\n")
	if len(r.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(r.Files))
	}
	if got := r.Files[0].Lines; got[1] != 3 || got[2] != 0 {
		t.Errorf("lines = %v", got)
	}
}

func TestParseIgnoresUnknownRecordsAndBlankLines(t *testing.T) {
	r := parse(t, "TN:name\n\nSF:a\nVER:1\nDA:1,1,checksum\nLF:99\nend_of_record\n")
	if got := r.File("a").Lines[1]; got != 1 {
		t.Errorf("hits = %d, want 1", got)
	}
}

func TestParseErrors(t *testing.T) {
	for name, in := range map[string]string{
		"data before SF":   "DA:1,1\n",
		"bad line number":  "SF:a\nDA:x,1\n",
		"missing hits":     "SF:a\nDA:1\n",
		"short branch":     "SF:a\nBRDA:1,0,0\n",
		"bad branch count": "SF:a\nBRDA:1,0,0,q\n",
		"negative count":   "SF:a\nDA:1,-3\n",
	} {
		if _, err := Parse(strings.NewReader(in)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestParseClampsHugeCounts(t *testing.T) {
	r := parse(t, "SF:a\nDA:1,99999999999999999999\nend_of_record\n")
	if r.File("a").Lines[1] <= 0 {
		t.Error("huge count should be clamped, not rejected or wrapped")
	}
}

func TestMergeSumsCounts(t *testing.T) {
	a := parse(t, "SF:a\nDA:1,1\nDA:2,0\nFN:1,f\nFNDA:1,f\nBRDA:1,0,0,1\nBRDA:1,0,1,0\nend_of_record\n")
	b := parse(t, "SF:a\nDA:2,4\nDA:3,1\nFN:1,f\nFNDA:2,f\nFN:5,g\nFNDA:0,g\nBRDA:1,0,0,-\nBRDA:1,0,1,3\nend_of_record\nSF:b\nDA:1,1\nend_of_record\n")
	m := Merge(a, nil, b)
	if len(m.Files) != 2 {
		t.Fatalf("files = %d, want 2", len(m.Files))
	}
	f := m.File("a")
	if !reflect.DeepEqual(f.Lines, map[int]int{1: 1, 2: 4, 3: 1}) {
		t.Errorf("lines = %v", f.Lines)
	}
	if !reflect.DeepEqual(f.Functions, []Function{{"f", 1, 3}, {"g", 5, 0}}) {
		t.Errorf("functions = %v", f.Functions)
	}
	if !reflect.DeepEqual(f.Branches, []Branch{{1, 0, 0, 1}, {1, 0, 1, 3}}) {
		t.Errorf("branches = %v", f.Branches)
	}
	if f.LineStatus(1) != Covered {
		t.Errorf("after merging, line 1 should be covered, got %v", f.LineStatus(1))
	}
}

func TestMergeDoesNotModifyInputs(t *testing.T) {
	a := parse(t, "SF:a\nDA:1,1\nend_of_record\n")
	b := parse(t, "SF:a\nDA:1,1\nend_of_record\n")
	Merge(a, b)
	if a.File("a").Lines[1] != 1 {
		t.Error("Merge modified its input")
	}
}

func TestRelPath(t *testing.T) {
	for _, tc := range []struct {
		path  string
		roots []string
		want  string
	}{
		{"file:///work/repo/pkg/a.ts", []string{"/work/repo"}, "pkg/a.ts"},
		{"/sandbox/tmp/pkg/a.ts", []string{"/work/repo", "/sandbox/tmp"}, "pkg/a.ts"},
		{"./pkg/../pkg/a.ts", nil, "pkg/a.ts"},
		{"/elsewhere/a.ts", []string{"/work/repo"}, "/elsewhere/a.ts"},
		{"/work/repository/a.ts", []string{"/work/repo"}, "/work/repository/a.ts"},
		{"pkg/a.ts", []string{"", "/work/repo"}, "pkg/a.ts"},
	} {
		if got := RelPath(tc.path, tc.roots...); got != tc.want {
			t.Errorf("RelPath(%q, %v) = %q, want %q", tc.path, tc.roots, got, tc.want)
		}
	}
}

func TestNormalizePathsMergesCollisions(t *testing.T) {
	r := parse(t, "SF:/w/a.ts\nDA:1,1\nend_of_record\nSF:a.ts\nDA:1,2\nend_of_record\n")
	r.NormalizePaths(func(p string) string { return RelPath(p, "/w") })
	if len(r.Files) != 1 || r.Files[0].Path != "a.ts" || r.Files[0].Lines[1] != 3 {
		t.Errorf("report = %+v", r.Files)
	}
}
