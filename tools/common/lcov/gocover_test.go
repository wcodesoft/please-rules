package lcov

import (
	"fmt"
	"strings"
	"testing"
)

// Source and profile (`go test -covermode=count -coverprofile`) of a small fixture:
// Classify(-1), Classify(5), Both(false, true) and OneLine(5) ran; Unused did not.
const goFixtureSource = `package fx

func Classify(n int) string {
	if n < 0 {
		return "neg"
	} else if n == 0 {
		return "zero"
	}
	return "pos"
}

func Both(a, b bool) bool {
	return a && b
}

func OneLine(x int) int {
	if x > 100 { return 100 }
	return x
}

func Unused() int {
	return 42
}

type T struct{}

func (t *T) Method() int {
	return 1
}

func (T) Empty() {}
`

const goFixtureProfile = `mode: count
example.com/fx/lib.go:3.29,4.11 1 2
example.com/fx/lib.go:4.11,6.3 1 1
example.com/fx/lib.go:6.8,6.19 1 1
example.com/fx/lib.go:6.19,8.3 1 0
example.com/fx/lib.go:9.2,9.14 1 1
example.com/fx/lib.go:12.27,14.2 1 1
example.com/fx/lib.go:16.25,17.13 1 1
example.com/fx/lib.go:17.13,17.27 1 0
example.com/fx/lib.go:18.2,18.10 1 1
example.com/fx/lib.go:21.19,23.2 1 0
example.com/fx/lib.go:27.26,29.2 1 3
`

func goFixture(t *testing.T) *File {
	t.Helper()
	r, err := FromGoProfile(strings.NewReader(goFixtureProfile),
		func(string) ([]byte, error) { return []byte(goFixtureSource), nil },
		func(p string) string { return strings.TrimPrefix(p, "example.com/fx/") })
	if err != nil {
		t.Fatal(err)
	}
	f := r.File("lib.go")
	if f == nil {
		t.Fatalf("lib.go missing, got %v", r.Files)
	}
	return f
}

func TestFromGoProfileFunctions(t *testing.T) {
	f := goFixture(t)
	want := []Function{
		{"Classify", 3, 2}, {"Both", 12, 1}, {"OneLine", 16, 1}, {"Unused", 21, 0}, {"T.Method", 27, 3},
	}
	if len(f.Functions) != len(want) {
		t.Fatalf("functions = %v, want %v (Empty has no statements, so no record)", f.Functions, want)
	}
	for i, w := range want {
		if f.Functions[i] != w {
			t.Errorf("function %d = %+v, want %+v", i, f.Functions[i], w)
		}
	}
}

func TestFromGoProfileLineStatuses(t *testing.T) {
	f := goFixture(t)
	for line, want := range map[int]Status{
		3:  Covered,       // function body opens
		4:  Covered,       // two blocks, both ran
		5:  Covered,       // "neg"
		6:  Partial,       // } else if n == 0 {  : the else-if body (a sub-line block) never ran
		7:  Uncovered,     // "zero"
		8:  Uncovered,     // closing brace of the block that never ran
		9:  Covered,       // return "pos"
		12: Covered,       // Go's coverage does not track short-circuit operators
		17: Partial,       // if x > 100 { return 100 } : the one-line body never ran
		18: Covered,       //
		21: Uncovered,     // function that never ran
		1:  NotExecutable, // no block
		10: NotExecutable,
	} {
		if got := f.LineStatus(line); got != want {
			t.Errorf("LineStatus(%d) = %v, want %v", line, got, want)
		}
	}
}

func TestFromGoProfileBranchRecords(t *testing.T) {
	f := goFixture(t)
	want := []Branch{{6, 0, 0, 1}, {6, 0, 1, 0}, {17, 0, 0, 0}}
	if len(f.Branches) != len(want) {
		t.Fatalf("branches = %v, want %v", f.Branches, want)
	}
	for i, w := range want {
		if f.Branches[i] != w {
			t.Errorf("branch %d = %+v, want %+v", i, f.Branches[i], w)
		}
	}
}

func TestFromGoProfileSetModeAndNoSource(t *testing.T) {
	r, err := FromGoProfile(strings.NewReader("mode: set\nexample.com/fx/lib.go:3.29,4.11 1 1\nexample.com/fx/lib.go:4.11,6.3 1 0\n"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := r.Files[0]
	if f.Path != "example.com/fx/lib.go" || len(f.Functions) != 0 {
		t.Errorf("without readSource and mapPath the path is kept and functions are skipped: %+v", f)
	}
	if f.Lines[3] != 1 || f.Lines[5] != 0 {
		t.Errorf("lines = %v", f.Lines)
	}
}

func TestFromGoProfileSourceProblemsAreNotFatal(t *testing.T) {
	for name, read := range map[string]func(string) ([]byte, error){
		"unreadable": func(string) ([]byte, error) { return nil, fmt.Errorf("no such file") },
		"unparsable": func(string) ([]byte, error) { return []byte("not go"), nil },
	} {
		r, err := FromGoProfile(strings.NewReader(goFixtureProfile), read, nil)
		if err != nil || len(r.Files[0].Functions) != 0 || len(r.Files[0].Lines) == 0 {
			t.Errorf("%s source: err=%v functions=%d lines=%d; want lines without functions", name, err, len(r.Files[0].Functions), len(r.Files[0].Lines))
		}
	}
}

func TestFromGoProfileMalformed(t *testing.T) {
	for name, in := range map[string]string{
		"no file":        "mode: set\nnot a block\n",
		"bad block":      "mode: set\na.go:1.2,3.4 x y\n",
		"truncated line": "mode: set\na.go:1.2,3.4 1\n",
	} {
		if _, err := FromGoProfile(strings.NewReader(in), nil, nil); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestFromGoProfileRoundTripsThroughLcov(t *testing.T) {
	f := goFixture(t)
	var out strings.Builder
	if err := (&Report{Files: []*File{f}}).Write(&out); err != nil {
		t.Fatal(err)
	}
	back, err := Parse(strings.NewReader(out.String()))
	if err != nil {
		t.Fatal(err)
	}
	g := back.File("lib.go")
	for line := 1; line <= 24; line++ {
		if f.LineStatus(line) != g.LineStatus(line) {
			t.Errorf("line %d: %v became %v after write and parse", line, f.LineStatus(line), g.LineStatus(line))
		}
	}
}

// coverage.py labels branch arms with text instead of numbers.
func TestParseTextBranchArms(t *testing.T) {
	r := parse(t, `SF:lib.py
DA:2,1
DA:4,1
BRDA:2,0,jump to line 3,1
BRDA:2,0,jump to line 4,1
BRDA:4,0,jump to line 5,0
BRDA:4,0,jump to line 7,1
end_of_record
`)
	f := r.File("lib.py")
	want := []Branch{{2, 0, 0, 1}, {2, 0, 1, 1}, {4, 0, 0, 0}, {4, 0, 1, 1}}
	if len(f.Branches) != len(want) {
		t.Fatalf("branches = %v, want %v", f.Branches, want)
	}
	for i, w := range want {
		if f.Branches[i] != w {
			t.Errorf("branch %d = %+v, want %+v", i, f.Branches[i], w)
		}
	}
	if f.LineStatus(2) != Covered || f.LineStatus(4) != Partial {
		t.Errorf("statuses: 2=%v 4=%v", f.LineStatus(2), f.LineStatus(4))
	}
}

func TestParseTextBranchArmsAreStablePerBranchPoint(t *testing.T) {
	// The same label on another line is a different arm; repeating a label reuses it.
	r := parse(t, "SF:a\nDA:1,1\nDA:2,1\nBRDA:1,0,x,1\nBRDA:2,0,x,1\nBRDA:2,0,y,0\nBRDA:2,0,x,3\nend_of_record\n")
	f := r.File("a")
	if len(f.Branches) != 3 {
		t.Fatalf("branches = %v", f.Branches)
	}
	if f.Branches[2] != (Branch{2, 0, 1, 0}) || f.Branches[1] != (Branch{2, 0, 0, 4}) {
		t.Errorf("repeated label should merge into arm 0 (1+3=4): %v", f.Branches)
	}
}
