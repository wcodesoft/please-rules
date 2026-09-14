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
