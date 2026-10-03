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
upstream `go-rules` and `python-rules` plugins, which do not write lcov. They
can still produce it without forking either plugin; see
[Go](#go-go_test-from-go-rules) and
[Python](#python-python_test-from-python-rules) below.

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

| Rule set                     | Line counts          | Functions                                            | Branches                                                                                     |
| :--------------------------- | :------------------- | :--------------------------------------------------- | :------------------------------------------------------------------------------------------- |
| ts (Deno, Vitest)            | real counts          | `FN`/`FNDA`                                          | real `BRDA` from V8 coverage                                                                 |
| kotlin                       | 0 or 1               | one per method, `<pkg>.<Class>.<method><descriptor>` | per line counts of arms that ran and did not; arms are anonymous (JaCoCo does not say which) |
| rust                         | real counts          | `FN`/`FNDA`, v0-mangled names (`_R...`)              | synthesized for partly executed lines (see below)                                            |
| swift                        | real counts          | `FN`/`FNDA`, Swift-mangled names, closures separate  | synthesized for partly executed lines (see below)                                            |
| go (converted)               | 0 or 1 (`mode: set`) | from parsing the source, `Type.Method`               | synthesized for partly executed lines; none for `a && b`                                     |
| python (`pytest-cov` recipe) | real counts          | `FN`/`FNDA` from coverage.py                         | real `BRDA`, arms labelled by text (`jump to line 5`) and numbered by the parser             |

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

## Go (`go_test` from go-rules)

`go_test` has no lcov output, but Please keeps the raw coverage file of every
target after `plz cover` at a hidden path, with no rule change:

```text
plz-out/bin/<package>/.test_coverage_<target name>
```

For a `go_test` this is the Go cover profile (`mode: set`, then
`file:3.29,4.11 1 1` block lines) written by `go tool covdata textfmt`. Convert
it with `gocover2lcov`:

```bash
./pleasew build //tools/common/gocover2lcov
plz-out/bin/tools/common/gocover2lcov/gocover2lcov \
  -profile plz-out/bin/pkg/.test_coverage_pkg_test \
  -src-root . -out plz-out/bin/pkg/pkg_test.lcov
```

Use `-strip-prefix <module path>` when the profile's file names are not
repository-relative (they are when the Go plugin's `ImportPath` is empty). The
library function is `lcov.FromGoProfile`.

- **Lines**: a line's count is the largest count of the blocks touching it. With
  `mode: set` counts are 0 or 1.
- **Functions**: the profile has none, so they come from parsing the source with
  `go/parser`: one record per function or method (`Type.Method`), with the count
  of the first block inside its body. Functions without statements get no
  record.
- **Partial lines**: a block that starts on a line after another block already
  covers the start of the line is a sub-line arm, such as the body of a one-line
  `if`. A line that ran with at least one never-run arm gets one `BRDA` record
  per arm, in column order. This is a heuristic, like the Rust and Swift one.
  Go's coverage has no notion of short-circuit operators, so `a && b` is never
  partial.
- **Hidden file**: `.test_coverage_<name>` is Please's own naming, not a
  documented interface, so check it in CI. Run `plz cover` on the package or a
  `...` pattern: a single explicit `//pkg:test` target printed `No data` in the
  console report in our tests even though the file was written.

---

## Python (`python_test` from python-rules)

The runner of `python-rules` writes line-only Cobertura XML (no methods, no
branch data), and its own collector cannot also produce lcov. `pytest-cov` can,
using only arguments `python_test` already has. This recipe was verified on
`python-rules` v2.1.3 with the `pytest` runner, `pytest-cov` 6.3 and coverage.py
7.10; it is not packaged as a rule.

Add `pytest-cov` and `coverage` as `pip_library` targets, make both part of the
runner dependencies (`TestrunnerDeps`, as `plz cover` needs `coverage` in the
pex anyway), and write a `coveragerc` next to the test:

```ini
[run]
branch = True
omit =
    */third_party/*
    */.bootstrap/*
    */test_*.py
```

```starlark
python_test(
    name = "lib_test",
    srcs = ["test_lib.py"],
    # The library as data puts its sources on disk, so paths are repository-relative.
    data = ["coveragerc", ":lib"],
    deps = [":lib", "//third_party/python:pytest_cov"],
    env = {
        "COVERAGE": "",                         # disable the runner's own collector
        "COVERAGE_FILE": ".coverage_pytest",    # coverage.py's data file, not Please's
        "PYTHONPATH": ".",                      # import the on-disk sources
    },
    flags = "--cov --cov-config=pkg/coveragerc --cov-report=lcov:lib_test.lcov --cov-report=xml:test.coverage",
    test_outputs = ["lib_test.lcov"],
)
```

Result in the test fixture: `plz cover` reports the same lines as the stock
runner (`lib.py 8/10`), and `plz-out/bin/pkg/lib_test.lcov` has `FN`/`FNDA`
records (`unused` at 0) and `BRDA` records.

- **Why the three variables**: `COVERAGE_FILE` is also coverage.py's own
  environment variable for its data file, so without the override `pytest-cov`
  overwrites the file Please reads and the target reports 0%. Without
  `PYTHONPATH` the library is imported from inside the pex, whose paths
  (`lib_test.pex/pkg/lib.py`) match no repository file; Please then also reports
  0%.
- **Branch arms are text**: coverage.py writes arms as `jump to line 5`. The
  parser numbers them in order of appearance per branch point.
- **Costs**: every run measures coverage, also a plain `plz test` (the flags are
  fixed at build time); the library sources are staged twice (data and pex); and
  `data` must name every library whose code should be measured.
- **Not covered here**: `unittest` and `behave` runners, and `zip_safe` or
  multi-interpreter variants.

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
