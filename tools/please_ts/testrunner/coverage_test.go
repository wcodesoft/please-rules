package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveCoverageMatrix(t *testing.T) {
	tests := []struct {
		name         string
		opts         RunOptions
		wantActive   bool
		wantFilename string
	}{
		{
			name: "explicit coverage enabled without file uses default",
			opts: RunOptions{
				Coverage: true,
			},
			wantActive:   true,
			wantFilename: "test.coverage",
		},
		{
			name: "explicit coverage enabled with custom file",
			opts: RunOptions{
				Coverage:     true,
				CoverageFile: "custom.coverage",
			},
			wantActive:   true,
			wantFilename: "custom.coverage",
		},
		{
			name: "coverage disabled",
			opts: RunOptions{
				Coverage: false,
			},
			wantActive:   false,
			wantFilename: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotActive, gotFile := tt.opts.resolveCoverage()
			if gotActive != tt.wantActive {
				t.Errorf("resolveCoverage active = %v, want %v", gotActive, tt.wantActive)
			}
			if gotFile != tt.wantFilename {
				t.Errorf("resolveCoverage file = %q, want %q", gotFile, tt.wantFilename)
			}
		})
	}
}

func TestLcovToCoberturaXMLMatrix(t *testing.T) {
	tests := []struct {
		name         string
		lcov         string
		cwd          string
		wantContains []string
	}{
		{
			name: "standard lcov record produces cobertura classes and line hits",
			lcov: `SF:/home/user/repo/test/ts/lib/calculator.ts
DA:1,1
DA:2,2
DA:5,0
end_of_record
`,
			cwd: "/home/user/repo",
			wantContains: []string{
				`<coverage>`,
				`filename="test/ts/lib/calculator.ts"`,
				`<line number="1" hits="1"/>`,
				`<line number="2" hits="2"/>`,
				`<line number="5" hits="0"/>`,
			},
		},
		{
			name: "empty lcov produces empty packages element",
			lcov: "",
			cwd:  "/home/user/repo",
			wantContains: []string{
				"<packages/>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xmlBytes := lcovToCoberturaXML([]byte(tt.lcov), tt.cwd)
			xmlStr := string(xmlBytes)

			for _, want := range tt.wantContains {
				if !strings.Contains(xmlStr, want) {
					t.Errorf("coverage XML missing %q in:\n%s", want, xmlStr)
				}
			}
		})
	}
}

func TestCoveragePath(t *testing.T) {
	cwd := "/run/dir"
	for in, want := range map[string]string{
		"file:///run/dir/pkg/lib.ts": "pkg/lib.ts",
		"/run/dir/lib.ts":            "lib.ts",
		"./pkg/lib.ts":               "pkg/lib.ts",
		// compiled layout: <pkg>/<name>/<name>.ts maps back to <pkg>/<name>.ts
		"pkg/lib/lib.ts": "pkg/lib.ts",
	} {
		if got := coveragePath(in, cwd); got != want {
			t.Errorf("coveragePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteRawLcovKeepsFunctionsAndBranches(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.lcov")
	in := "SF:/run/dir/lib.ts\nFN:1,f\nFNDA:0,f\nDA:2,0\nBRDA:2,0,0,-\nend_of_record\n"
	if err := writeRawLcov(out, []byte(in), "/run/dir"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	for _, want := range []string{"SF:lib.ts\n", "FN:1,f\n", "FNDA:0,f\n", "BRDA:2,0,0,-\n", "FNF:1\n", "BRF:1\n"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestWriteRawLcovDisabledAndInvalid(t *testing.T) {
	if err := writeRawLcov("", []byte("SF:a\n"), "/x"); err != nil {
		t.Errorf("empty path must be a no-op, got %v", err)
	}
	if err := writeRawLcov(filepath.Join(t.TempDir(), "o"), []byte("DA:1,1\n"), "/x"); err == nil {
		t.Error("expected an error for a record outside an SF section")
	}
}

func TestRunCreatesEmptyLcovFileWithoutCoverage(t *testing.T) {
	file := filepath.Join(t.TempDir(), "target.lcov")
	if err := os.WriteFile(file, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}
	// No sources: Run fails, but only after validating its options; use the
	// helper that Run uses to prepare the output instead.
	opts := RunOptions{LcovFile: file}
	if err := opts.prepareLcovFile(); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(file); len(data) != 0 {
		t.Errorf("lcov file should be reset to empty, got %q", data)
	}
}
