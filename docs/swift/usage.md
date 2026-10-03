# Swift Rules Usage Guide

This guide walks through common scenarios and patterns for using the Swift rules
in your Please repository.

---

## 1. Project Setup

Configure your root `.plzconfig` to register the Swift plugin and enable code
coverage:

```ini
[Plugin "swift"]
Target = //plugins:swift
SwiftcTool = swiftc
LlvmProfdataTool = llvm-profdata
LlvmCovTool = llvm-cov

[cover]
fileextension = .swift
```

---

## 2. Multi-Module Project Structure

A typical multi-module Swift project layout:

```txt
my-project/
├── .plzconfig
├── src/
│   ├── core/
│   │   ├── BUILD
│   │   └── Core.swift
│   ├── network/
│   │   ├── BUILD
│   │   ├── Network.swift
│   │   └── NetworkTests.swift
│   └── cli/
│       ├── BUILD
│       └── main.swift
```

### `src/core/BUILD`

```python
swift_library(
    name = "core",
    srcs = ["Core.swift"],
    module_name = "Core",
    visibility = ["PUBLIC"],
)
```

### `src/network/BUILD`

```python
swift_library(
    name = "network",
    srcs = ["Network.swift"],
    module_name = "Network",
    deps = ["//src/core:core"],
    visibility = ["PUBLIC"],
)

swift_test(
    name = "network_test",
    srcs = ["NetworkTests.swift"],
    deps = [":network"],
)
```

### `src/cli/BUILD`

```python
swift_binary(
    name = "cli",
    srcs = ["main.swift"],
    deps = ["//src/network:network"],
)
```

---

## 3. Writing Modern Tests with `swift-testing`

`swift_test` defaults to `swift-testing` (`import Testing`).

Example test file (`MathTests.swift`):

```swift
import Testing
import Math

@Suite("Math Suite")
struct MathTests {
    @Test("Basic addition")
    func add() {
        #expect(Math.add(1, 2) == 3)
    }

    @Test("Parameterized test", arguments: [
        (2, 2, 4),
        (3, 5, 8),
        (10, -5, 5),
    ])
    func parameterizedAdd(a: Int, b: Int, expected: Int) {
        #expect(Math.add(a, b) == expected)
    }
}
```

Run tests:

```bash
./pleasew test //src/network:network_test
```

---

## 4. Measuring Code Coverage

Run tests with Please's built-in coverage runner:

```bash
./pleasew cover //src/network:network_test
```

Please will display line coverage percentages in the terminal and write detailed
reports to `plz-out/log/coverage.json` and `plz-out/log/coverage.xml`.

---

## 5. Hermetic Toolchains

For hermetic and reproducible builds across developer machines and CI, assemble
an official Swift release toolchain using `swift_toolchain`:

```python
# tools/toolchains/BUILD
swift_toolchain(
    name = "swift_toolchain",
    version = "6.3.3",
    visibility = ["PUBLIC"],
)
```

Then configure `.plzconfig` to point to the toolchain targets:

```ini
[Plugin "swift"]
Target = //plugins:swift
SwiftcTool = //tools/toolchains:swift_toolchain|bin/swiftc
LlvmProfdataTool = //tools/toolchains:swift_toolchain|bin/llvm-profdata
LlvmCovTool = //tools/toolchains:swift_toolchain|bin/llvm-cov
```

---

## Code Coverage (`plz cover`)

### Required `[cover]` setting

Please only reports coverage for files whose extension is listed under `[cover]`
in your `.plzconfig`. The Swift plugin cannot set this for you; without it the
report is silently `Total coverage: No data`, or lists files of other languages
instead of yours.

```ini
[cover]
fileextension = .swift
```

If your repository mixes languages, add one `fileextension` line per extension.

### Running

```bash
./pleasew cover //src/...
```

Under `cover`, the test runner recompiles the test binary with
`-profile-generate -profile-coverage-mapping`, runs it, and exports the LLVM
profile with `llvm-profdata` and `llvm-cov`.

### Report output

- **Location**: Please merges the per-test results into
  `plz-out/log/coverage.xml` (Cobertura XML) and `plz-out/log/coverage.json`
  (per-file line markers plus per-file and per-directory percentages).
- **Contents**: line coverage only. Function and branch data are in the raw lcov
  export described below.
- **Format**: LLVM lcov converted to Cobertura XML in `$COVERAGE_FILE`.

### Raw lcov export (functions and branches)

Each `swift_test` also keeps the `llvm-cov` report as lcov, with the function
(`FN`/`FNDA`) and branch (`BRDA`) records that `coverage.xml` and
`coverage.json` do not carry:

- **Location**: `plz-out/bin/<package>/<target name>.lcov`, for example
  `plz-out/bin/test/swift/branches/branches_test.lcov`. It is declared as a test
  output, so it is written next to the test binary.
- **Content**: lcov with repository-relative paths; the synthetic test runner is
  excluded. The file is empty unless the test ran under `plz cover`.
- **Functions**: `FN`/`FNDA` records with Swift-mangled names (demangle with
  `swift demangle`). Closures are separate records: the right-hand side of
  `a && b` is an autoclosure that shows up as its own function with 0 hits when
  it never ran.
- **Partial lines**: Swift emits no branch regions, so `llvm-cov` has no branch
  records of its own. Instead, a `BRDA` record is synthesized for every line
  that ran but contains a sub-line region that never did, such as the body of an
  `else if` that begins on a line with other code, or the right-hand side of
  `a && b`: one arm per region starting on the line, in column order, with its
  count. This is a heuristic, not source-level branches. If the arms of a branch
  are on separate lines, the untaken side shows up as an uncovered line, not as
  a partial one. Lines whose sub-line regions all ran carry no branch record.
- **Lifetime**: the file reflects the last test run that actually executed. Read
  it right after `./pleasew cover --rerun <target>`; do not assume it is
  recreated when Please reuses a cached result.
