package linkorder

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeLib(t *testing.T, root, name string, deps ...string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	meta := `{"module_name":"` + name + `","dependencies":[`
	for i, d := range deps {
		if i > 0 {
			meta += ","
		}
		meta += `"` + d + `"`
	}
	meta += "]}"
	if err := os.WriteFile(filepath.Join(dir, "swift_metadata.json"), []byte(meta), 0644); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "lib"+name+".a")
}

func TestSortPutsDependentsBeforeDependencies(t *testing.T) {
	root := t.TempDir()
	base := writeLib(t, root, "Base")
	mid := writeLib(t, root, "Mid", "Base")
	top := writeLib(t, root, "Top", "Mid", "Base")

	got := Sort([]string{base, mid, top})
	want := []string{top, mid, base}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Sort = %v, want %v", got, want)
	}
}

func TestSortDiamondAndCycleTerminate(t *testing.T) {
	root := t.TempDir()
	d := writeLib(t, root, "D")
	b := writeLib(t, root, "B", "D")
	c := writeLib(t, root, "C", "D")
	a := writeLib(t, root, "A", "B", "C")
	got := Sort([]string{d, b, c, a})
	pos := map[string]int{}
	for i, p := range got {
		pos[filepath.Base(p)] = i
	}
	if !(pos["libA.a"] < pos["libB.a"] && pos["libA.a"] < pos["libC.a"] && pos["libB.a"] < pos["libD.a"] && pos["libC.a"] < pos["libD.a"]) {
		t.Errorf("bad diamond order: %v", got)
	}

	x := writeLib(t, root, "X", "Y")
	y := writeLib(t, root, "Y", "X")
	if got := Sort([]string{x, y}); len(got) != 2 {
		t.Errorf("cycle lost archives: %v", got)
	}
}

func TestSortWithoutMetadataKeepsInput(t *testing.T) {
	in := []string{"a/liba.a", "b/libb.a"}
	if got := Sort(in); !reflect.DeepEqual(got, in) {
		t.Errorf("Sort = %v, want %v", got, in)
	}
}
