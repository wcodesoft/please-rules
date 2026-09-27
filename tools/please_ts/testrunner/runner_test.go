package testrunner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFallbackJUnitMatrix(t *testing.T) {
	tests := []struct {
		name         string
		srcs         []string
		testErr      error
		wantFailures int
		wantContains []string
	}{
		{
			name:         "failed test execution creates failure element",
			srcs:         []string{"foo_test.ts", "bar_test.ts"},
			testErr:      errors.New("sample execution error"),
			wantFailures: 1,
			wantContains: []string{
				"<failure message=\"sample execution error\">",
				"failures=\"1\"",
				"tests=\"2\"",
			},
		},
		{
			name:         "successful test execution has zero failures",
			srcs:         []string{"sample_test.ts"},
			testErr:      nil,
			wantFailures: 0,
			wantContains: []string{
				"failures=\"0\"",
				"tests=\"1\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "fallback_junit_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			resultsFile := filepath.Join(tmpDir, "test.results")
			if err := writeFallbackJUnit(resultsFile, tt.srcs, tt.testErr); err != nil {
				t.Fatalf("writeFallbackJUnit failed: %v", err)
			}

			data, err := os.ReadFile(resultsFile)
			if err != nil {
				t.Fatal(err)
			}
			content := string(data)

			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Errorf("content missing %q in:\n%s", want, content)
				}
			}
		})
	}
}

func TestBoolToIntMatrix(t *testing.T) {
	tests := []struct {
		input bool
		want  int
	}{
		{true, 1},
		{false, 0},
	}

	for _, tt := range tests {
		t.Run(string(rune('0'+tt.want)), func(t *testing.T) {
			got := boolToInt(tt.input)
			if got != tt.want {
				t.Errorf("boolToInt(%v) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
