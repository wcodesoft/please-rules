package testrunner

import (
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
