package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBrowserJUnitMatrix(t *testing.T) {
	tests := []struct {
		name         string
		results      []browserTestResult
		wantFailures int
		wantPasses   int
		wantErr      bool
	}{
		{
			name: "all browser tests pass",
			results: []browserTestResult{
				{Name: "test 1", Duration: 0.05, Error: ""},
				{Name: "test 2", Duration: 0.10, Error: ""},
			},
			wantFailures: 0,
			wantPasses:   2,
			wantErr:      false,
		},
		{
			name: "browser test with failure returns error and writes failure xml",
			results: []browserTestResult{
				{Name: "passing test", Duration: 0.02, Error: ""},
				{Name: "failing test", Duration: 0.04, Error: "assertion failed"},
			},
			wantFailures: 1,
			wantPasses:   1,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "browser_junit_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			resultsFile := filepath.Join(tmpDir, "test.results")
			err = writeBrowserJUnit(resultsFile, []string{"browser_test.ts"}, tt.results)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeBrowserJUnit() error = %v, wantErr = %v", err, tt.wantErr)
			}

			data, err := os.ReadFile(resultsFile)
			if err != nil {
				t.Fatal(err)
			}
			content := string(data)

			if !strings.Contains(content, "<testsuites") {
				t.Errorf("expected <testsuites> in output: %s", content)
			}
			if tt.wantFailures > 0 && !strings.Contains(content, "<failure message=\"test failed\">") {
				t.Errorf("expected failure element in output: %s", content)
			}
		})
	}
}

func TestEscapeXMLMatrix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello & world", "hello &amp; world"},
		{"<script>", "&lt;script&gt;"},
		{`"quoted"`, "&#34;quoted&#34;"},
		{"plain text", "plain text"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeXML(tt.input)
			if got != tt.want {
				t.Errorf("escapeXML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
