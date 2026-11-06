package testrunner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// snapshotHelperJS adds expectComputedStyles to the page. It records the computed style of every
// element matching a selector into window.__SNAPSHOTS__; the host compares the recording with the
// golden file once the tests have finished, so the helper itself never fails a test on a diff.
const snapshotHelperJS = `(() => {
	window.__SNAPSHOTS__ = {};
	window.expectComputedStyles = function(name, selector, props) {
		if (!/^[A-Za-z0-9_.-]+$/.test(name)) {
			throw new Error('invalid snapshot name ' + JSON.stringify(name) + ': use letters, digits, "_", "." and "-"');
		}
		if (name in window.__SNAPSHOTS__) {
			throw new Error('snapshot ' + name + ' recorded twice');
		}
		if (!props || props.length === 0) {
			throw new Error('snapshot ' + name + ' needs at least one property');
		}
		const els = Array.from(document.querySelectorAll(selector));
		if (els.length === 0) {
			throw new Error('snapshot ' + name + ': no element matches ' + selector);
		}
		window.__SNAPSHOTS__[name] = els.map((el, index) => {
			const cs = window.getComputedStyle(el);
			const styles = {};
			for (const p of props) {
				styles[p] = cs.getPropertyValue(p);
			}
			return { selector: selector, index: index, styles: styles };
		});
	};
})()`

// validSnapshotName reports whether name is non-empty and made of letters, digits, "_", "." and "-",
// so it can never escape the snapshot directory.
func validSnapshotName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.', r == '-':
		default:
			return false
		}
	}
	return name != "." && name != ".."
}

// snapshotElement is the computed style of one element matched by a snapshot selector.
type snapshotElement struct {
	Selector string            `json:"selector"`
	Index    int               `json:"index"`
	Styles   map[string]string `json:"styles"`
}

// parseViewport reads a "WIDTHxHEIGHT" viewport. The empty string means no override.
func parseViewport(s string) (width, height int, err error) {
	if s == "" {
		return 0, 0, nil
	}
	w, h, ok := strings.Cut(s, "x")
	if !ok {
		return 0, 0, fmt.Errorf("invalid viewport %q: want WIDTHxHEIGHT", s)
	}
	if width, err = strconv.Atoi(w); err != nil || width <= 0 {
		return 0, 0, fmt.Errorf("invalid viewport %q: want WIDTHxHEIGHT", s)
	}
	if height, err = strconv.Atoi(h); err != nil || height <= 0 {
		return 0, 0, fmt.Errorf("invalid viewport %q: want WIDTHxHEIGHT", s)
	}
	return width, height, nil
}

// canonicalSnapshot renders elements as stable, indented JSON (map keys are sorted).
func canonicalSnapshot(elements []snapshotElement) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(elements); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// diffSnapshots lists the differences between a golden and an actual recording, one per line.
func diffSnapshots(golden, actual []snapshotElement) []string {
	var diffs []string
	if len(golden) != len(actual) {
		diffs = append(diffs, fmt.Sprintf("matched %d elements, golden has %d", len(actual), len(golden)))
	}
	for i := 0; i < len(golden) && i < len(actual); i++ {
		g, a := golden[i], actual[i]
		where := fmt.Sprintf("%s[%d]", g.Selector, g.Index)
		if g.Selector != a.Selector {
			diffs = append(diffs, fmt.Sprintf("%s: selector is now %q", where, a.Selector))
			continue
		}
		props := map[string]bool{}
		for p := range g.Styles {
			props[p] = true
		}
		for p := range a.Styles {
			props[p] = true
		}
		names := make([]string, 0, len(props))
		for p := range props {
			names = append(names, p)
		}
		sort.Strings(names)
		for _, p := range names {
			gv, gok := g.Styles[p]
			av, aok := a.Styles[p]
			switch {
			case !gok:
				diffs = append(diffs, fmt.Sprintf("%s %s: new property, value %q", where, p, av))
			case !aok:
				diffs = append(diffs, fmt.Sprintf("%s %s: no longer recorded, golden %q", where, p, gv))
			case gv != av:
				diffs = append(diffs, fmt.Sprintf("%s %s: golden %q, actual %q", where, p, gv, av))
			}
		}
	}
	return diffs
}

// checkSnapshots compares the recordings against the golden files and returns one test result
// per snapshot. Every recording is also written to outDir (when set), so accepting a change means
// copying that file over the golden. A golden that nothing recorded is reported as stale.
func checkSnapshots(goldenFiles []string, outDir string, recorded map[string][]snapshotElement) ([]browserTestResult, error) {
	goldens := map[string]string{}
	for _, f := range goldenFiles {
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		if !validSnapshotName(name) {
			return nil, fmt.Errorf("invalid golden file name %q", f)
		}
		if prev, dup := goldens[name]; dup {
			return nil, fmt.Errorf("goldens %q and %q share the snapshot name %q", prev, f, name)
		}
		goldens[name] = f
	}

	var results []browserTestResult
	for _, name := range sortedKeys(recorded) {
		if !validSnapshotName(name) {
			return nil, fmt.Errorf("invalid snapshot name %q", name)
		}
		actual := recorded[name]
		actualJSON, err := canonicalSnapshot(actual)
		if err != nil {
			return nil, err
		}
		if outDir != "" {
			if err := os.MkdirAll(outDir, 0755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(outDir, name+".json"), actualJSON, 0644); err != nil {
				return nil, err
			}
		}

		res := browserTestResult{Name: "snapshot: " + name}
		goldenPath, ok := goldens[name]
		if !ok {
			res.Error = fmt.Sprintf("no golden for snapshot %q; add %s.json to the target's snapshots (the recording is in the test outputs)", name, name)
			results = append(results, res)
			continue
		}
		delete(goldens, name)

		raw, err := os.ReadFile(goldenPath)
		if err != nil {
			return nil, err
		}
		var golden []snapshotElement
		if err := json.Unmarshal(raw, &golden); err != nil {
			res.Error = fmt.Sprintf("golden %s is not valid: %v", goldenPath, err)
			results = append(results, res)
			continue
		}
		if diffs := diffSnapshots(golden, actual); len(diffs) > 0 {
			res.Error = fmt.Sprintf("snapshot %q differs from %s:\n  %s", name, goldenPath, strings.Join(diffs, "\n  "))
		}
		results = append(results, res)
	}

	for _, name := range sortedKeys(goldens) {
		results = append(results, browserTestResult{
			Name:  "snapshot: " + name,
			Error: fmt.Sprintf("golden %s was not recorded by any test; delete it or restore the test", goldens[name]),
		})
	}
	return results, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
