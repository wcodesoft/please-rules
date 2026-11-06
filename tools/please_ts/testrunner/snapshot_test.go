package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseViewport(t *testing.T) {
	tests := []struct {
		in           string
		wantW, wantH int
		wantErr      bool
	}{
		{in: ""},
		{in: "1280x800", wantW: 1280, wantH: 800},
		{in: "1280", wantErr: true},
		{in: "0x800", wantErr: true},
		{in: "axb", wantErr: true},
		{in: "1280x-1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			w, h, err := parseViewport(tt.in)
			if (err != nil) != tt.wantErr || w != tt.wantW || h != tt.wantH {
				t.Errorf("parseViewport(%q) = %d, %d, %v", tt.in, w, h, err)
			}
		})
	}
}

func TestValidSnapshotName(t *testing.T) {
	for name, want := range map[string]bool{
		"button":      true,
		"nav-bar_2.x": true,
		"":            false,
		"..":          false,
		"a/b":         false,
		"../up":       false,
		"has space":   false,
	} {
		if got := validSnapshotName(name); got != want {
			t.Errorf("validSnapshotName(%q) = %v, want %v", name, got, want)
		}
	}
}

func snap(selector string, index int, styles map[string]string) snapshotElement {
	return snapshotElement{Selector: selector, Index: index, Styles: styles}
}

func TestDiffSnapshots(t *testing.T) {
	base := []snapshotElement{snap(".b", 0, map[string]string{"color": "red", "margin": "0px"})}
	tests := []struct {
		name   string
		actual []snapshotElement
		want   []string
	}{
		{name: "identical", actual: base},
		{
			name:   "changed value",
			actual: []snapshotElement{snap(".b", 0, map[string]string{"color": "blue", "margin": "0px"})},
			want:   []string{`.b[0] color: golden "red", actual "blue"`},
		},
		{
			name:   "new and dropped property",
			actual: []snapshotElement{snap(".b", 0, map[string]string{"color": "red", "padding": "1px"})},
			want: []string{
				`.b[0] margin: no longer recorded, golden "0px"`,
				`.b[0] padding: new property, value "1px"`,
			},
		},
		{
			name:   "element count",
			actual: append(append([]snapshotElement{}, base...), snap(".b", 1, map[string]string{"color": "red", "margin": "0px"})),
			want:   []string{"matched 2 elements, golden has 1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffSnapshots(base, tt.actual)
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Errorf("diffSnapshots() = %q, want %q", got, tt.want)
			}
		})
	}
}

func writeGolden(t *testing.T, dir, name string, elements []snapshotElement) string {
	t.Helper()
	data, err := canonicalSnapshot(elements)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckSnapshots(t *testing.T) {
	red := []snapshotElement{snap(".b", 0, map[string]string{"color": "red"})}
	blue := []snapshotElement{snap(".b", 0, map[string]string{"color": "blue"})}

	tests := []struct {
		name       string
		goldens    map[string][]snapshotElement
		recorded   map[string][]snapshotElement
		wantErrors map[string]string // result name -> substring of its error, "" for a pass
	}{
		{
			name:       "match passes",
			goldens:    map[string][]snapshotElement{"a": red},
			recorded:   map[string][]snapshotElement{"a": red},
			wantErrors: map[string]string{"snapshot: a": ""},
		},
		{
			name:       "mismatch fails with a diff",
			goldens:    map[string][]snapshotElement{"a": red},
			recorded:   map[string][]snapshotElement{"a": blue},
			wantErrors: map[string]string{"snapshot: a": `golden "red", actual "blue"`},
		},
		{
			name:       "missing golden fails",
			recorded:   map[string][]snapshotElement{"a": red},
			wantErrors: map[string]string{"snapshot: a": "no golden"},
		},
		{
			name:       "stale golden fails",
			goldens:    map[string][]snapshotElement{"a": red, "old": red},
			recorded:   map[string][]snapshotElement{"a": red},
			wantErrors: map[string]string{"snapshot: a": "", "snapshot: old": "not recorded"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goldenDir, outDir := t.TempDir(), filepath.Join(t.TempDir(), "out")
			var files []string
			for name, els := range tt.goldens {
				files = append(files, writeGolden(t, goldenDir, name, els))
			}

			results, err := checkSnapshots(files, outDir, tt.recorded)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != len(tt.wantErrors) {
				t.Fatalf("got %d results %v, want %d", len(results), results, len(tt.wantErrors))
			}
			for _, r := range results {
				want, ok := tt.wantErrors[r.Name]
				if !ok {
					t.Errorf("unexpected result %q", r.Name)
				} else if want == "" && r.Error != "" || !strings.Contains(r.Error, want) {
					t.Errorf("%s: error %q, want it to contain %q", r.Name, r.Error, want)
				}
			}
			for name := range tt.recorded {
				if _, err := os.Stat(filepath.Join(outDir, name+".json")); err != nil {
					t.Errorf("recording %s missing from the outputs: %v", name, err)
				}
			}
		})
	}
}

func TestCheckSnapshotsRejectsUnsafeNames(t *testing.T) {
	if _, err := checkSnapshots(nil, t.TempDir(), map[string][]snapshotElement{"../evil": nil}); err == nil {
		t.Error("a snapshot named ../evil was accepted")
	}
	if _, err := checkSnapshots([]string{"a/b c.json"}, "", nil); err == nil {
		t.Error("a golden named 'b c' was accepted")
	}
}
