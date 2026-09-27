package importmap

import (
	"testing"
)

func TestFindTargetEntryMatrix(t *testing.T) {
	tests := []struct {
		name       string
		moduleName string
		srcs       []string
		want       string
	}{
		{
			name:       "exact match with module base name",
			moduleName: "@domain/calculator",
			srcs:       []string{"add.ts", "calculator.ts", "subtract.ts"},
			want:       "calculator.ts",
		},
		{
			name:       "exact match with tsx extension",
			moduleName: "Button",
			srcs:       []string{"Button.tsx", "styles.css"},
			want:       "Button.tsx",
		},
		{
			name:       "index.ts preferred when module name file is absent",
			moduleName: "@repo/utils",
			srcs:       []string{"format.ts", "index.ts", "math.ts"},
			want:       "index.ts",
		},
		{
			name:       "mod.ts preferred when index is absent",
			moduleName: "std_wrapper",
			srcs:       []string{"helper.ts", "mod.ts"},
			want:       "mod.ts",
		},
		{
			name:       "fallback to first source file when no candidate matches",
			moduleName: "random_lib",
			srcs:       []string{"first.ts", "second.ts"},
			want:       "first.ts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findTargetEntry(tt.moduleName, tt.srcs)
			if got != tt.want {
				t.Errorf("findTargetEntry(%q, %v) = %q, want %q", tt.moduleName, tt.srcs, got, tt.want)
			}
		})
	}
}

func TestIsSourceFileMatrix(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"index.ts", true},
		{"component.tsx", true},
		{"bundle.js", true},
		{"app.jsx", true},
		{"mod.mjs", true},
		{"data.json", false},
		{"main.go", false},
		{"README.md", false},
		{"build.sh", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isSourceFile(tt.path)
			if got != tt.want {
				t.Errorf("isSourceFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDirFromPathMatrix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"./foo/bar.js", "./foo/"},
		{"foo/bar.js", "./foo/"},
		{"./calculator.ts", "./"},
		{"calculator.ts", "./"},
		{"./third_party/react/index.js", "./third_party/react/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := dirFromPath(tt.input)
			if got != tt.want {
				t.Errorf("dirFromPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEnsureTrailingSlashMatrix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"third_party/react", "./third_party/react/"},
		{"./third_party/react", "./third_party/react/"},
		{"./third_party/react/", "./third_party/react/"},
		{"/root/path", "/root/path/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ensureTrailingSlash(tt.input)
			if got != tt.want {
				t.Errorf("ensureTrailingSlash(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
