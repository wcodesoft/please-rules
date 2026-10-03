package lcov

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Parse reads an lcov tracefile. Several records for the same file are merged.
// Unknown record types (TN, FNF, LF, BRH, ...) are ignored: summary counts are
// recomputed when writing.
func Parse(r io.Reader) (*Report, error) {
	report := &Report{}
	byPath := make(map[string]*File)
	// fnIndex finds a function record by name within the current record, since
	// FNDA lines refer to FN lines by name.
	var cur *File
	fnIndex := map[string]int{}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		key, val, _ := strings.Cut(line, ":")
		fail := func(err error) (*Report, error) {
			return nil, fmt.Errorf("lcov line %d: %s: %w", lineNo, key, err)
		}

		switch key {
		case "SF":
			f, ok := byPath[val]
			if !ok {
				f = newFile(val)
				byPath[val] = f
				report.Files = append(report.Files, f)
			}
			cur = f
			fnIndex = map[string]int{}
			for i, fn := range f.Functions {
				fnIndex[fn.Name] = i
			}
		case "end_of_record":
			cur = nil
		case "DA":
			if cur == nil {
				return fail(errOutsideRecord)
			}
			parts := strings.Split(val, ",")
			if len(parts) < 2 {
				return fail(errFields)
			}
			n, err1 := strconv.Atoi(parts[0])
			hits, err2 := parseCount(parts[1])
			if err1 != nil || err2 != nil {
				return fail(errNumber)
			}
			cur.Lines[n] += hits
		case "FN":
			if cur == nil {
				return fail(errOutsideRecord)
			}
			// FN:<line>,<name> or FN:<start>,<end>,<name>; names may contain commas.
			parts := strings.SplitN(val, ",", 3)
			if len(parts) < 2 {
				return fail(errFields)
			}
			n, err := strconv.Atoi(parts[0])
			if err != nil {
				return fail(errNumber)
			}
			name := parts[1]
			if len(parts) == 3 {
				if _, err := strconv.Atoi(parts[1]); err == nil {
					name = parts[2]
				} else {
					name = parts[1] + "," + parts[2]
				}
			}
			if _, dup := fnIndex[name]; !dup {
				fnIndex[name] = len(cur.Functions)
				cur.Functions = append(cur.Functions, Function{Name: name, Line: n})
			}
		case "FNDA":
			if cur == nil {
				return fail(errOutsideRecord)
			}
			hitsStr, name, ok := strings.Cut(val, ",")
			if !ok {
				return fail(errFields)
			}
			hits, err := parseCount(hitsStr)
			if err != nil {
				return fail(errNumber)
			}
			if i, ok := fnIndex[name]; ok {
				cur.Functions[i].Hits += hits
			}
		case "BRDA":
			if cur == nil {
				return fail(errOutsideRecord)
			}
			parts := strings.Split(val, ",")
			if len(parts) != 4 {
				return fail(errFields)
			}
			ln, e1 := strconv.Atoi(parts[0])
			block, e2 := strconv.Atoi(parts[1])
			arm, e3 := strconv.Atoi(parts[2])
			if e1 != nil || e2 != nil || e3 != nil {
				return fail(errNumber)
			}
			taken := NotEvaluated
			if parts[3] != "-" {
				t, err := parseCount(parts[3])
				if err != nil {
					return fail(errNumber)
				}
				taken = t
			}
			cur.Branches = append(cur.Branches, Branch{Line: ln, Block: block, Arm: arm, Taken: taken})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for _, f := range report.Files {
		f.Branches = mergeBranches(f.Branches)
		f.sortRecords()
	}
	return report, nil
}

var (
	errOutsideRecord = fmt.Errorf("record outside an SF section")
	errFields        = fmt.Errorf("wrong number of fields")
	errNumber        = fmt.Errorf("invalid number")
)

// parseCount parses an execution count. Some tools write very large counts, which
// are clamped rather than rejected.
func parseCount(s string) (int, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
			return int(^uint(0) >> 1), nil
		}
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("negative count")
	}
	return int(n), nil
}
