package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/common/lcov"
)

// llvm-cov lcov export for a small fixture (Swift 6.3.3, no branch regions):
// classify(-1), classify(5) and both(false, true) ran; unused never did. The
// synthetic test runner entry point is included, as it is in a real export.
const swiftLcovExport = `SF:/work/plz-out/tmp/pkg/t._test/run_1/pkg/Lib.swift
FN:1,$s1t8classifyySSSiF
FN:11,$s1t4bothyS2b_SbtF
FN:12,Lib.swift:$s1t4bothyS2b_SbtFSbyKXEfu_
FN:15,$s1t6unusedSiyF
FNDA:2,$s1t8classifyySSSiF
FNDA:1,$s1t4bothyS2b_SbtF
FNDA:0,Lib.swift:$s1t4bothyS2b_SbtFSbyKXEfu_
FNDA:0,$s1t6unusedSiyF
DA:1,2
DA:2,2
DA:3,1
DA:4,1
DA:5,0
DA:6,1
DA:7,1
DA:8,1
DA:9,2
DA:11,1
DA:12,1
DA:13,1
DA:15,0
end_of_record
SF:/work/plz-out/tmp/pkg/t._test/run_1/__runner_main.swift
FN:1,main
FNDA:1,main
DA:1,1
end_of_record
`

// The matching JSON export: line 4 is "} else if n == 0 {" whose body never ran, and the
// right side of a && b on line 12 is a closure that never ran.
const swiftJSONExport = `{"data":[{"files":[{"filename":"/work/plz-out/tmp/pkg/t._test/run_1/pkg/Lib.swift","segments":[
[1,42,2,true,true,false],[2,8,2,true,true,false],[2,13,2,true,false,false],[2,14,1,true,true,false],
[4,6,2,true,false,false],[4,15,1,true,true,false],[4,21,2,true,false,false],[4,22,0,true,true,false],
[6,6,2,true,false,false],[6,12,1,true,true,false],[8,6,2,true,false,false],[9,2,0,false,false,false],
[11,48,1,true,true,false],[12,17,0,true,true,false],[12,18,1,true,false,false],[13,2,0,false,false,false]]}]}]}`

func TestBuildRawLcov(t *testing.T) {
	data, err := buildRawLcov([]byte(swiftLcovExport), []byte(swiftJSONExport), "/work")
	if err != nil {
		t.Fatal(err)
	}
	report, err := lcov.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Files) != 1 {
		t.Fatalf("files = %d, want the synthetic runner dropped", len(report.Files))
	}
	f := report.File("pkg/Lib.swift")
	if f == nil {
		t.Fatalf("pkg/Lib.swift missing, got %q", report.Files[0].Path)
	}
	if len(f.Functions) != 4 || f.FunctionsHit() != 2 {
		t.Errorf("functions = %v", f.Functions)
	}
	for line, want := range map[int]lcov.Status{
		4:  lcov.Partial,   // body of the untaken else-if
		12: lcov.Partial,   // closure for the right side of && never ran
		5:  lcov.Uncovered, // body of the untaken arm
		15: lcov.Uncovered,
		2:  lcov.Covered,
		6:  lcov.Covered,
	} {
		if got := f.LineStatus(line); got != want {
			t.Errorf("LineStatus(%d) = %v, want %v", line, got, want)
		}
	}
}

func TestBuildRawLcovWithoutRegionData(t *testing.T) {
	data, err := buildRawLcov([]byte(swiftLcovExport), nil, "/work")
	if err != nil {
		t.Fatal(err)
	}
	report, _ := lcov.Parse(strings.NewReader(string(data)))
	f := report.File("pkg/Lib.swift")
	if f == nil || len(f.Branches) != 0 || f.LineStatus(12) != lcov.Covered {
		t.Errorf("without the JSON export the report must still be written, lines only: %+v", f)
	}
}

func TestBuildRawLcovInvalidInput(t *testing.T) {
	if _, err := buildRawLcov([]byte("DA:1,1\n"), nil, "/work"); err == nil {
		t.Error("expected an error for a record outside an SF section")
	}
	if _, err := buildRawLcov([]byte(swiftLcovExport), []byte("not json"), "/work"); err == nil {
		t.Error("expected an error for an invalid JSON export")
	}
}

func TestCoveragePath(t *testing.T) {
	for in, want := range map[string]string{
		"/w/plz-out/tmp/p/t._test/run_1/p/Lib.swift": "p/Lib.swift",
		"/w/plz-out/tmp/p/t._build/p/Lib.swift":      "p/Lib.swift",
		"file:///w/p/Lib.swift":                      "p/Lib.swift",
		"./p/Lib.swift":                              "p/Lib.swift",
		"/x/__runner_main.swift":                     "",
	} {
		if got := coveragePath(in, "/w"); got != want {
			t.Errorf("coveragePath(%q) = %q, want %q", in, got, want)
		}
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
	if err := writeRawLcov(path, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); len(got) != 0 {
		t.Errorf("expected an empty file, got %q", got)
	}
}

// A fake llvm-cov that prints the JSON export, to test exportRawLcov end to end.
func fakeLlvmCov(t *testing.T, json string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "export.json"), []byte(json), 0644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "llvm-cov")
	body := "#!/bin/sh\ncat \"" + filepath.Join(dir, "export.json") + "\"\n"
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

func TestExportRawLcovUsesTheJSONExport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "t.lcov")
	opts := RunOptions{LlvmCov: fakeLlvmCov(t, swiftJSONExport), LcovFile: out}
	if err := exportRawLcov(opts, "bin", "x.profdata", []byte(swiftLcovExport), "/work"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	for _, want := range []string{"SF:pkg/Lib.swift\n", "BRDA:4,0,0,1\n", "BRDA:4,0,1,0\n", "BRDA:12,0,0,0\n"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestExportRawLcovFallsBackWhenTheJSONExportFails(t *testing.T) {
	out := filepath.Join(t.TempDir(), "t.lcov")
	opts := RunOptions{LlvmCov: "/nonexistent/llvm-cov", LcovFile: out}
	if err := exportRawLcov(opts, "bin", "x.profdata", []byte(swiftLcovExport), "/work"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if !strings.Contains(string(got), "SF:pkg/Lib.swift\n") || strings.Contains(string(got), "BRDA:") {
		t.Errorf("expected the lines-and-functions report without branches:\n%s", got)
	}
}
