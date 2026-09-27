package importmap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSynthesizeMetadataDepsMatrix(t *testing.T) {
	tests := []struct {
		name        string
		depRelDir   string
		metaFile    string
		metaJSON    string
		files       map[string]string
		appSrc      string
		appContent  string
		wantImports map[string]string
	}{
		{
			name:      "module metadata with entry point",
			depRelDir: "third_party/preact",
			metaFile:  "ts_module.json",
			metaJSON: `{
				"name": "preact",
				"entry": "dist/preact.mjs",
				"types": "dist/preact.d.ts"
			}`,
			files: map[string]string{
				"dist/preact.mjs": "export const h = () => {};",
			},
			appSrc:     "src/app.ts",
			appContent: "import { h } from 'preact';",
			wantImports: map[string]string{
				"preact":  "./third_party/preact/dist/preact.mjs",
				"preact/": "./third_party/preact/dist/",
			},
		},
		{
			name:      "library metadata with multiple subpath files",
			depRelDir: "plz-out/gen/components",
			metaFile:  "ts_metadata.json",
			metaJSON: `{
				"name": "@repo/dashboard/components",
				"entry": "Badge.ts",
				"files": ["Badge.ts", "MetricCard.ts"]
			}`,
			files: map[string]string{
				"Badge.ts":      "export const Badge = 'badge';",
				"MetricCard.ts": "export const MetricCard = 'card';",
			},
			appSrc:     "src/app.ts",
			appContent: "import { Badge } from '@repo/dashboard/components/Badge';",
			wantImports: map[string]string{
				"@repo/dashboard/components/Badge":         "./plz-out/gen/components/Badge.ts",
				"@repo/dashboard/components/Badge.ts":      "./plz-out/gen/components/Badge.ts",
				"@repo/dashboard/components/MetricCard":    "./plz-out/gen/components/MetricCard.ts",
				"@repo/dashboard/components/MetricCard.ts": "./plz-out/gen/components/MetricCard.ts",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "importmap_depmeta_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			depDir := filepath.Join(tmpDir, tt.depRelDir)
			_ = os.MkdirAll(depDir, 0755)

			for relPath, content := range tt.files {
				p := filepath.Join(depDir, relPath)
				_ = os.MkdirAll(filepath.Dir(p), 0755)
				_ = os.WriteFile(p, []byte(content), 0644)
			}
			_ = os.WriteFile(filepath.Join(depDir, tt.metaFile), []byte(tt.metaJSON), 0644)

			appPath := filepath.Join(tmpDir, tt.appSrc)
			_ = os.MkdirAll(filepath.Dir(appPath), 0755)
			_ = os.WriteFile(appPath, []byte(tt.appContent), 0644)

			im, err := Synthesize("", []string{appPath}, []string{depDir}, tmpDir)
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
