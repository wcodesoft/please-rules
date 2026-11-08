package bundle

import (
	"strings"
	"testing"

	"tools/please_ts/importmap"
)

func TestOptionsValidation(t *testing.T) {
	err := Run(Options{})
	if err == nil {
		t.Errorf("expected error when Main is empty")
	}

	err = Run(Options{Main: "main.ts"})
	if err == nil {
		t.Errorf("expected error when Out is empty")
	}
}

func TestBuildEsbuildArgs(t *testing.T) {
	opts := Options{
		Main:      "src/index.ts",
		Out:       "dist/bundle.js",
		Format:    "esm",
		Minify:    true,
		Sourcemap: true,
		Flags:     []string{"--target=esnext"},
	}

	args := buildEsbuildArgs(opts, nil)
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "src/index.ts") {
		t.Errorf("expected entry point in args: %s", argsStr)
	}
	if !strings.Contains(argsStr, "--outfile=dist/bundle.js") {
		t.Errorf("expected outfile in args: %s", argsStr)
	}
	if !strings.Contains(argsStr, "--format=esm") {
		t.Errorf("expected format in args: %s", argsStr)
	}
	if !strings.Contains(argsStr, "--minify") {
		t.Errorf("expected minify in args: %s", argsStr)
	}
	if !strings.Contains(argsStr, "--sourcemap") {
		t.Errorf("expected sourcemap in args: %s", argsStr)
	}
	if !strings.Contains(argsStr, "--target=esnext") {
		t.Errorf("expected extra flags in args: %s", argsStr)
	}
}

func TestBuildEsbuildArgsWithImportMap(t *testing.T) {
	im := &importmap.ImportMap{
		Imports: map[string]string{
			"preact":       "./third_party/npm/preact/index.js",
			"preact/hooks": "./third_party/npm/preact/hooks/index.js",
		},
	}
	opts := Options{
		Main: "src/index.ts",
		Out:  "dist/bundle.js",
	}
	args := buildEsbuildArgs(opts, im)
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--alias:preact=./third_party/npm/preact/index.js") {
		t.Errorf("expected preact alias in args: %s", argsStr)
	}
	if !strings.Contains(argsStr, "--alias:preact/hooks=./third_party/npm/preact/hooks/index.js") {
		t.Errorf("expected preact/hooks alias in args: %s", argsStr)
	}
}

func TestBuildEsbuildArgsFiltersInvalidAliases(t *testing.T) {
	im := &importmap.ImportMap{
		Imports: map[string]string{
			"./relative.ts":       "./test/relative.ts",
			"../parent.ts":        "../parent.ts",
			"/absolute.ts":        "/absolute.ts",
			"https://deno.land/x": "./cache/deno.js",
			"valid-pkg":           "./lib/index.js",
			"valid-pkg/":          "./lib/",
			"@scope/components":   "./components/index.ts",
			"@scope/components/":  "./components/",
		},
	}
	opts := Options{
		Main: "main.ts",
		Out:  "bundle.js",
	}
	args := buildEsbuildArgs(opts, im)
	argsStr := strings.Join(args, " ")

	if strings.Contains(argsStr, "--alias:./") || strings.Contains(argsStr, "--alias:../") || strings.Contains(argsStr, "--alias:/") {
		t.Errorf("relative or absolute paths should not be used as alias names: %s", argsStr)
	}
	if strings.Contains(argsStr, "--alias:https:") {
		t.Errorf("URL specifiers should not be used as alias names: %s", argsStr)
	}
	if !strings.Contains(argsStr, "--alias:valid-pkg=./lib/index.js") {
		t.Errorf("expected valid-pkg to point to entry file: %s", argsStr)
	}
	if !strings.Contains(argsStr, "--alias:@scope/components=./components/index.ts") {
		t.Errorf("expected @scope/components to point to entry file: %s", argsStr)
	}
}
