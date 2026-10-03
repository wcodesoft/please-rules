package lcov

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"sort"
	"strings"
)

type goBlock struct {
	startLine, startCol, endLine, endCol int
	count                                int
}

// FromGoProfile converts a Go cover profile (`go test -coverprofile`, any mode) into
// an lcov report with lines, functions and synthesized branch records.
//
// readSource returns the source of a profile path (needed for function records, which
// the profile does not have; may be nil to skip them) and mapPath turns the profile's
// import-path-style name into the report path (nil keeps it).
//
// Lines: a line's count is the largest count of the blocks touching it. Partial
// lines: a block that starts on a line after another block already covers the start
// of the line is a sub-line arm; a line that ran with at least one zero arm gets one
// BRDA record per arm, in column order (the same heuristic as AddRegionBranches).
// Go's coverage has no notion of short-circuit operators, so `a && b` never shows up
// as partial.
func FromGoProfile(profile io.Reader, readSource func(string) ([]byte, error), mapPath func(string) string) (*Report, error) {
	byFile := map[string][]goBlock{}
	var order []string
	sc := bufio.NewScanner(profile)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		file, rest, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("profile line %d: no file", n)
		}
		var b goBlock
		var stmts int
		if _, err := fmt.Sscanf(rest, "%d.%d,%d.%d %d %d", &b.startLine, &b.startCol, &b.endLine, &b.endCol, &stmts, &b.count); err != nil {
			return nil, fmt.Errorf("profile line %d: %w", n, err)
		}
		if _, seen := byFile[file]; !seen {
			order = append(order, file)
		}
		byFile[file] = append(byFile[file], b)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	report := &Report{}
	for _, name := range order {
		blocks := byFile[name]
		sort.Slice(blocks, func(i, j int) bool {
			if blocks[i].startLine != blocks[j].startLine {
				return blocks[i].startLine < blocks[j].startLine
			}
			return blocks[i].startCol < blocks[j].startCol
		})
		path := name
		if mapPath != nil {
			path = mapPath(name)
		}
		f := newFile(path)

		touching := map[int][]goBlock{}
		for _, b := range blocks {
			for l := b.startLine; l <= b.endLine; l++ {
				touching[l] = append(touching[l], b)
				if b.count > f.Lines[l] {
					f.Lines[l] = b.count
				} else if _, ok := f.Lines[l]; !ok {
					f.Lines[l] = b.count
				}
			}
		}

		for l, hits := range f.Lines {
			if hits == 0 {
				continue
			}
			var starting []goBlock
			coveredBefore := false
			for _, b := range touching[l] {
				if b.startLine == l {
					starting = append(starting, b)
				} else {
					coveredBefore = true
				}
			}
			if !coveredBefore && len(starting) > 0 {
				starting = starting[1:] // the line's own block
			}
			missed := false
			for _, b := range starting {
				if b.count == 0 {
					missed = true
				}
			}
			if missed {
				for arm, b := range starting {
					f.Branches = append(f.Branches, Branch{Line: l, Arm: arm, Taken: b.count})
				}
			}
		}

		if readSource != nil {
			if src, err := readSource(name); err == nil {
				f.Functions = goFunctions(src, blocks)
			}
		}
		f.sortRecords()
		report.Files = append(report.Files, f)
	}
	return report, nil
}

// goFunctions lists the functions and methods of src. A function's count is the count
// of the first block inside its body; functions without statements have no block and
// are left out.
func goFunctions(src []byte, blocks []goBlock) []Function {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil
	}
	var fns []Function
	for _, d := range file.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		open := fset.Position(fd.Body.Lbrace)
		closing := fset.Position(fd.Body.Rbrace)
		name := fd.Name.Name
		if fd.Recv != nil && len(fd.Recv.List) == 1 {
			name = goReceiver(fd.Recv.List[0].Type) + "." + name
		}
		for _, b := range blocks {
			afterOpen := b.startLine > open.Line || (b.startLine == open.Line && b.startCol >= open.Column)
			beforeClose := b.startLine < closing.Line || (b.startLine == closing.Line && b.startCol <= closing.Column)
			if afterOpen && beforeClose {
				fns = append(fns, Function{Name: name, Line: fset.Position(fd.Pos()).Line, Hits: b.count})
				break
			}
		}
	}
	return fns
}

func goReceiver(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return goReceiver(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return goReceiver(t.X)
	}
	return "?"
}
