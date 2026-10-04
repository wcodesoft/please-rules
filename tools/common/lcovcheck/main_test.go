package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The report Please kept for a Kotlin fixture: classify has an else-if whose second arm never
// ran (line 6), both() is short-circuited (line 14) and unused() is never called.
const kotlinLcov = `TN:
SF:test/kotlin/branches/branches.kt
FN:4,test.kotlin.branches.BranchesKt.classify(I)Ljava/lang/String;
FN:14,test.kotlin.branches.BranchesKt.both(ZZ)Z
FN:18,test.kotlin.branches.BranchesKt.unused()I
FNDA:1,test.kotlin.branches.BranchesKt.classify(I)Ljava/lang/String;
FNDA:1,test.kotlin.branches.BranchesKt.both(ZZ)Z
FNDA:0,test.kotlin.branches.BranchesKt.unused()I
FNF:3
FNH:2
BRDA:4,0,0,1
BRDA:4,0,1,1
BRDA:6,0,0,1
BRDA:6,0,1,0
BRDA:14,0,0,1
BRDA:14,0,1,0
BRF:6
BRH:4
DA:4,1
DA:5,1
DA:6,1
DA:7,0
DA:9,1
DA:14,1
DA:18,0
LF:7
LH:5
end_of_record
`

// The coverage.json Please wrote for the same run.
const coverageJSON = `{
    "tests": {},
    "files": {"test/kotlin/branches/branches.kt": "NNCNCUNNNNNCNNNNU"},
    "stats": {
        "total_coverage": 71.4,
        "coverage_by_file": {
            "test/kotlin/branches/branches.kt": 71.4,
            "test/kotlin/branches/untested.kt": 0
        },
        "coverage_by_directory": {"test/kotlin/branches": 71.4}
    }
}`

func write(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func check(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := run(args, &out)
	return out.String(), err
}

const src = "test/kotlin/branches/branches.kt"

func TestLcovAssertionsThatHold(t *testing.T) {
	file := write(t, "x.lcov", kotlinLcov)
	out, err := check(t, "-lcov", file, "-path", src,
		"-line", "4=covered", "-line", "5=covered", "-line", "6=partial", "-line", "7=uncovered", "-line", "8=not-executable",
		"-function", "classify=hit", "-function", "unused()I=missed", "-function", "both=1", "-function", "classify=1")
	if err != nil || !strings.Contains(out, "all assertions hold") {
		t.Fatalf("out = %q, err = %v", out, err)
	}
}

func TestLcovFailuresAreAllListed(t *testing.T) {
	file := write(t, "x.lcov", kotlinLcov)
	_, err := check(t, "-lcov", file, "-path", src,
		"-line", "6=covered", // really partial
		"-line", "7=partial", // really uncovered
		"-function", "unused=hit",
		"-function", "classify=2")
	if err == nil {
		t.Fatal("expected failures")
	}
	for _, want := range []string{
		"4 assertion(s) failed",
		"line 6 is partial, want covered",
		"line 7 is uncovered, want partial",
		"function test.kotlin.branches.BranchesKt.unused()I: was never called, want it called",
		"was called 1 times, want 2",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q:\n%v", want, err)
		}
	}
}

func TestMissedAssertionOnACalledFunction(t *testing.T) {
	file := write(t, "x.lcov", kotlinLcov)
	_, err := check(t, "-lcov", file, "-path", src, "-function", "classify=missed")
	if err == nil || !strings.Contains(err.Error(), "was called 1 times, want it never called") {
		t.Errorf("got %v", err)
	}
}

func TestFunctionNamesMustIdentifyExactlyOneFunction(t *testing.T) {
	file := write(t, "x.lcov", kotlinLcov)
	for _, substr := range []string{"BranchesKt", "nothing_like_this"} {
		_, err := check(t, "-lcov", file, "-path", src, "-function", substr+"=hit")
		if err == nil || !strings.Contains(err.Error(), "want exactly 1") {
			t.Errorf("%q: got %v", substr, err)
		}
	}
}

func TestMissingSourceFileListsWhatTheReportHas(t *testing.T) {
	file := write(t, "x.lcov", kotlinLcov)
	_, err := check(t, "-lcov", file, "-path", "other/file.kt", "-line", "1=covered")
	if err == nil || !strings.Contains(err.Error(), "other/file.kt is not in the report") || !strings.Contains(err.Error(), src) {
		t.Errorf("got %v", err)
	}
}

func TestBadAssertionsAreReported(t *testing.T) {
	file := write(t, "x.lcov", kotlinLcov)
	_, err := check(t, "-lcov", file, "-path", src, "-line", "x=covered", "-line", "4=maybe", "-line", "4", "-function", "=hit", "-function", "classify=often")
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, want := range []string{`bad -line "x=covered"`, `bad -line "4=maybe"`, `bad -line "4"`, `bad -function "=hit"`, `bad expectation "often"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q:\n%v", want, err)
		}
	}
}

func TestCoverageJSON(t *testing.T) {
	file := write(t, "coverage.json", coverageJSON)
	if out, err := check(t, "-coverage-json", file, "-covered", src); err != nil || !strings.Contains(out, "all assertions hold") {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	_, err := check(t, "-coverage-json", file, "-covered", src, "-covered", "test/kotlin/branches/untested.kt", "-covered", "test/gone.kt")
	if err == nil || !strings.Contains(err.Error(), "2 assertion(s) failed") ||
		!strings.Contains(err.Error(), "untested.kt has no coverage (0.0%)") || !strings.Contains(err.Error(), "test/gone.kt is not in the coverage report") {
		t.Errorf("got %v", err)
	}
}

func TestUsageErrors(t *testing.T) {
	lcovFile := write(t, "x.lcov", kotlinLcov)
	jsonFile := write(t, "c.json", coverageJSON)
	for name, args := range map[string][]string{
		"nothing given":           {},
		"both modes":              {"-lcov", lcovFile, "-coverage-json", jsonFile},
		"lcov without path":       {"-lcov", lcovFile, "-line", "4=covered"},
		"lcov without assertions": {"-lcov", lcovFile, "-path", src},
		"json without files":      {"-coverage-json", jsonFile},
		"missing lcov file":       {"-lcov", "/nonexistent.lcov", "-path", src, "-line", "4=covered"},
		"missing json file":       {"-coverage-json", "/nonexistent.json", "-covered", src},
		"invalid json":            {"-coverage-json", write(t, "bad.json", "{"), "-covered", src},
		"invalid lcov":            {"-lcov", write(t, "bad.lcov", "DA:1,1\n"), "-path", src, "-line", "1=covered"},
		"unknown flag":            {"-nope"},
	} {
		if _, err := check(t, args...); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
