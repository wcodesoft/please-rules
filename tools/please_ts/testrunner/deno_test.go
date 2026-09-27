package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildNodePathMatrix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nodepath_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	depDir1 := filepath.Join(tmpDir, "dep1")
	_ = os.MkdirAll(depDir1, 0755)
	depDir2 := filepath.Join(tmpDir, "dep2")
	_ = os.MkdirAll(depDir2, 0755)

	// Create a simulated module with package.json
	subModuleDir := filepath.Join(tmpDir, "modules", "sample_pkg")
	_ = os.MkdirAll(subModuleDir, 0755)
	_ = os.WriteFile(filepath.Join(subModuleDir, "package.json"), []byte("{}"), 0644)

	tests := []struct {
		name         string
		deps         []string
		wantContains []string
	}{
		{
			name: "directories included in node path",
			deps: []string{depDir1, depDir2},
			wantContains: []string{
				depDir1,
				depDir2,
			},
		},
		{
			name: "explicit dep with package.json discovers module root",
			deps: []string{subModuleDir},
			wantContains: []string{
				subModuleDir,
				filepath.Dir(subModuleDir),
			},
		},
		{
			name:         "empty deps includes working directory",
			deps:         nil,
			wantContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNodePath(tt.deps)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("buildNodePath missing %q in %q", want, got)
				}
			}
		})
	}
}

func TestResolveSourcesMatrix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resolve_srcs_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	f1 := filepath.Join(tmpDir, "calc_test.ts")
	_ = os.WriteFile(f1, []byte(""), 0644)

	tests := []struct {
		name string
		srcs []string
		want []string
	}{
		{
			name: "exact existing file is preserved",
			srcs: []string{f1},
			want: []string{f1},
		},
		{
			name: "non-existent file is returned as-is when walk finds nothing",
			srcs: []string{"missing_file.ts"},
			want: []string{"missing_file.ts"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSources(tt.srcs)
			if len(got) != len(tt.want) {
				t.Fatalf("resolveSources len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("resolveSources[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
