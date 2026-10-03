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
- **Contents**: line coverage only. Branch coverage and per-function data are
  not emitted, so consumers cannot join coverage with per-function complexity
  from this report.
- **Format**: LLVM lcov converted to Cobertura XML in `$COVERAGE_FILE`.
