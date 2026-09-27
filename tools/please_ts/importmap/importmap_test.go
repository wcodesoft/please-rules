package importmap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSynthesizeTargetSourcesMatrix(t *testing.T) {
	tests := []struct {
		name        string
		moduleName  string
		files       map[string]string
		srcs        []string
		wantImports map[string]string
	}{
		{
			name:       "single entry file matches module name",
			moduleName: "@domain/calculator",
			files: map[string]string{
				"calculator.ts": "export function add(a: number, b: number) { return a + b; }",
			},
			srcs: []string{"calculator.ts"},
			wantImports: map[string]string{
				"@domain/calculator":               "./calculator.ts",
				"@domain/calculator/":              "./",
				"@domain/calculator/calculator":    "./calculator.ts",
				"@domain/calculator/calculator.ts": "./calculator.ts",
			},
		},
		{
			name:       "multiple source files with subpaths",
			moduleName: "@repo/dashboard/components",
			files: map[string]string{
				"Badge.ts":      "export const Badge = 'badge';",
				"MetricCard.ts": "export const MetricCard = 'card';",
				"Modal.ts":      "export const Modal = 'modal';",
			},
			srcs: []string{"Badge.ts", "MetricCard.ts", "Modal.ts"},
			wantImports: map[string]string{
				"@repo/dashboard/components/Badge":         "./Badge.ts",
				"@repo/dashboard/components/Badge.ts":      "./Badge.ts",
				"@repo/dashboard/components/MetricCard":    "./MetricCard.ts",
				"@repo/dashboard/components/MetricCard.ts": "./MetricCard.ts",
				"@repo/dashboard/components/Modal":         "./Modal.ts",
				"@repo/dashboard/components/Modal.ts":      "./Modal.ts",
			},
		},
		{
			name:       "empty module name skips root mapping",
			moduleName: "",
			files: map[string]string{
				"app.ts": "console.log('hello');",
			},
			srcs:        []string{"app.ts"},
			wantImports: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "importmap_target_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			var absSrcs []string
			for relPath, content := range tt.files {
				fullPath := filepath.Join(tmpDir, relPath)
				_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			for _, s := range tt.srcs {
				absSrcs = append(absSrcs, filepath.Join(tmpDir, s))
			}

			im, err := Synthesize(tt.moduleName, absSrcs, nil, tmpDir)
			if err != nil {
				t.Fatalf("Synthesize failed: %v", err)
			}

			for key, want := range tt.wantImports {
				got, ok := im.Imports[key]
				if !ok {
					t.Errorf("import[%q] missing, want %q", key, want)
				} else if got != want {
					t.Errorf("import[%q] = %q, want %q", key, got, want)
				}
			}
		})
	}
}
