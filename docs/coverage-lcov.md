# Raw lcov Coverage Reports

`plz cover` reduces every language's coverage to per-line hit counts in
`plz-out/log/coverage.xml` and `coverage.json`. The test rules of this
repository also keep the coverage tool's own report as **lcov**, which carries
what the merged report drops: function records and branch records. This page is
the contract for consumers, such as an annotated source view or a coverage
dashboard.

---

## Where the files are

Each test target writes one file, declared as a test output:

```text
plz-out/bin/<package>/<target name>.lcov
```

| Rule                      | Language   | Example                                               |
| :------------------------ | :--------- | :---------------------------------------------------- |
| `ts_test`                 | TypeScript | `plz-out/bin/test/ts/lib/branches_test.lcov`          |
| `kotlin_test` (`kt_test`) | Kotlin     | `plz-out/bin/test/kotlin/branches/branches_test.lcov` |
| `rust_test`               | Rust       | `plz-out/bin/test/rust/branches/branches_test.lcov`   |
| `swift_test`              | Swift      | `plz-out/bin/test/swift/branches/branches_test.lcov`  |

Find them with `plz-out/bin/**/*.lcov`. Go and Python tests come from the
upstream `go-rules` and `python-rules` plugins, which do not write lcov: Go
writes its native block profile and Python writes line-only Cobertura XML to
`$COVERAGE_FILE`, where Please reads it.

### Lifetime

- The file is only filled when the test runs under `plz cover`. A plain
  `plz test` leaves an empty file.
- It reflects the last test run that actually executed. Produce fresh data with
  `./pleasew cover --rerun <targets>` and read the files right after.
- Do not assume the file is recreated when Please reuses a cached result: a
  deleted file stays deleted. Treat a missing or empty file as "no data for this
  target", not as "no coverage".

---

## Format

Standard lcov tracefile, one record per source file:

| Record                 | Meaning                                                                                 |
| :--------------------- | :-------------------------------------------------------------------------------------- |
| `SF:<path>`            | Source file, **repository-relative** and slash-separated                                |
| `FN:<line>,<n>`        | Function `n` starting at `line`                                                         |
| `FNDA:<c>,<n>`         | Function `n` ran `c` times                                                              |
| `DA:<line>,<c>`        | Executable line ran `c` times (`0` = never)                                             |
| `BRDA:<l>,<b>,<a>,<t>` | Branch arm `a` of block `b` on line `l`, taken `t` times, or `-` if the block never ran |

Lines without a `DA` record are not executable. Summary records (`LF`, `LH`,
`FNF`, `FNH`, `BRF`, `BRH`) are written for convenience and can be recomputed.

### What each rule set provides

| Rule set          | Line counts | Functions                                            | Branches                                                                                     |
| :---------------- | :---------- | :--------------------------------------------------- | :------------------------------------------------------------------------------------------- |
| ts (Deno, Vitest) | real counts | `FN`/`FNDA`                                          | real `BRDA` from V8 coverage                                                                 |
| kotlin            | 0 or 1      | one per method, `<pkg>.<Class>.<method><descriptor>` | per line counts of arms that ran and did not; arms are anonymous (JaCoCo does not say which) |
| rust              | real counts | `FN`/`FNDA`, v0-mangled names (`_R...`)              | synthesized for partly executed lines (see below)                                            |
| swift             | real counts | `FN`/`FNDA`, Swift-mangled names, closures separate  | synthesized for partly executed lines (see below)                                            |

**Synthesized branches (Rust, Swift).** Neither compiler emits branch regions on
stable, so the branch records are derived from `llvm-cov`'s sub-line regions: a
line that ran but contains a region that never did (the right-hand side of
`a && b`, the body of a one-line `if`) gets one `BRDA` arm per region starting
on that line, in column order. This is a heuristic. A branch whose arms are on
separate lines shows up as an uncovered line, not as a partial one, and lines
whose sub-line regions all ran carry no `BRDA` record.

**Do not compare branch arm numbers across tools.** Two tools number the arms of
the same branch point differently (the Deno and Vitest runners do). Merge
records of the same file only when they come from the same tool.

---

## Classifying a line

For an annotated source view, classify each source line from its records:

| Status             | Rule                                                                |
| :----------------- | :------------------------------------------------------------------ |
| **Not executable** | no `DA` record for the line                                         |
| **Uncovered**      | `DA` count is `0`                                                   |
| **Partial**        | `DA` count is above `0` and a `BRDA` arm on the line has `0` or `-` |
| **Covered**        | `DA` count is above `0` and no arm on the line was missed           |

Function coverage comes from `FNDA` (above `0` = ran); for tools without
function records, join the line statuses with the function's line range.

Show the source of the same revision the tests ran on, otherwise the line
numbers drift.

---

## Reading and merging

The reference implementation is the Go package
[`tools/common/lcov`](../tools/common/lcov), used by the test runners to write
these files. It provides:

- `lcov.Parse` and `Report.Write`: read and write tracefiles (output is sorted
  and deterministic; summary records are recomputed);
- `lcov.Merge`: sum the counts of several reports, for example the reports of
  several test targets. Lines match by number, functions by name, branches by
  `(line, block, arm)`. A line is covered after merging if any target ran it,
  and the merged report cannot say which target covered it;
- `File.LineStatus`: the classification above;
- `RelPath` and `Report.NormalizePaths`: repository-relative paths.

The format is plain lcov, so a consumer outside this repository can use any lcov
parser and apply the rules above.
