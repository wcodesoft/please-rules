package lcov

// Merge combines reports, for example the reports of several test targets, into a
// new one. Execution counts are summed:
//
//   - lines are matched by number;
//   - functions by name (the line of the first report that has the function wins);
//   - branches by (line, block, arm). A NotEvaluated arm merged with a counted one
//     takes the count.
//
// Branch identifiers are only comparable between reports produced by the same tool:
// two tools numbering the arms of one branch point differently (the Deno and Vitest
// runners do) must not be merged for the same file, or arms will be paired wrongly.
func Merge(reports ...*Report) *Report {
	out := &Report{}
	byPath := make(map[string]*File)
	for _, r := range reports {
		if r == nil {
			continue
		}
		for _, f := range r.Files {
			dst, ok := byPath[f.Path]
			if !ok {
				dst = newFile(f.Path)
				byPath[f.Path] = dst
				out.Files = append(out.Files, dst)
			}
			for n, hits := range f.Lines {
				dst.Lines[n] += hits
			}
			for _, fn := range f.Functions {
				found := false
				for i := range dst.Functions {
					if dst.Functions[i].Name == fn.Name {
						dst.Functions[i].Hits += fn.Hits
						found = true
						break
					}
				}
				if !found {
					dst.Functions = append(dst.Functions, fn)
				}
			}
			dst.Branches = append(dst.Branches, f.Branches...)
		}
	}
	for _, f := range out.Files {
		f.Branches = mergeBranches(f.Branches)
		f.sortRecords()
	}
	return out
}

// mergeBranches sums branches that share (line, block, arm).
func mergeBranches(in []Branch) []Branch {
	type key struct{ line, block, arm int }
	index := make(map[key]int, len(in))
	var out []Branch
	for _, b := range in {
		k := key{b.Line, b.Block, b.Arm}
		i, ok := index[k]
		if !ok {
			index[k] = len(out)
			out = append(out, b)
			continue
		}
		switch {
		case out[i].Taken == NotEvaluated:
			out[i].Taken = b.Taken
		case b.Taken != NotEvaluated:
			out[i].Taken += b.Taken
		}
	}
	return out
}
