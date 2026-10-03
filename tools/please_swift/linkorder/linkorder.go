// Package linkorder orders static archives for linking.
package linkorder

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type metadata struct {
	ModuleName   string   `json:"module_name"`
	Dependencies []string `json:"dependencies"`
}

// Sort orders archives so that every archive precedes the archives of the modules it
// depends on, according to the swift_metadata.json next to each archive. GNU gold
// resolves symbols from archives in command-line order and does not honour
// --start-group here, so a library listed after its dependent would be dropped.
// Archives without metadata keep their relative position.
func Sort(archives []string) []string {
	modules := make(map[string]int) // module name -> index in archives
	deps := make([][]string, len(archives))
	for i, a := range archives {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(a), "swift_metadata.json"))
		if err != nil {
			continue
		}
		var m metadata
		if json.Unmarshal(data, &m) != nil || m.ModuleName == "" {
			continue
		}
		modules[m.ModuleName] = i
		deps[i] = m.Dependencies
	}

	// Depth-first post-order puts dependencies before dependents; reversing it
	// puts dependents first.
	visited := make([]bool, len(archives))
	var post []int
	var visit func(i int)
	visit = func(i int) {
		if visited[i] {
			return
		}
		visited[i] = true
		for _, d := range deps[i] {
			if j, ok := modules[d]; ok {
				visit(j)
			}
		}
		post = append(post, i)
	}
	// Visit in reverse so that unrelated archives keep their input order once the
	// post-order is reversed below.
	for i := len(archives) - 1; i >= 0; i-- {
		visit(i)
	}

	sorted := make([]string, 0, len(archives))
	for k := len(post) - 1; k >= 0; k-- {
		sorted = append(sorted, archives[post[k]])
	}
	return sorted
}
