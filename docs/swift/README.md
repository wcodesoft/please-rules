# Swift Rules for Please (`///swift`)

Hermetic, performant, and idiomatic [Swift](https://www.swift.org/) build rules
for the [Please](https://please.build) build system.

Designed for modern Swift 6+, these rules provide seamless support for
multi-module libraries, standalone binaries, modern test suites powered by
`swift-testing`, and first-class code coverage.

---

## Features

- **⚡ Fast Incremental Builds**: Module-level compilation caching via Please
  and `swiftc`.
- **🧪 Modern Testing**: Native support for modern `swift-testing`
  (`import Testing`, `@Test`, `#expect(...)`) as well as legacy `XCTest`,
  producing standard JUnit XML reports.
- **📊 First-Class Code Coverage**: Seamless integration with `pleasew cover`,
  utilizing `llvm-profdata` and `llvm-cov` to generate standard Cobertura XML
  with exact line-hit metrics.
- **📦 Multi-Module Dependencies**: Automatic Swift module search paths (`-I`),
  interface handling (`.swiftmodule`, `.swiftdoc`, `.abi.json`), and static
  archive linking (`.a`).
- **🔒 Hermetic Toolchains**: Supports system compiler discovery or fully
  hermetic Swift toolchains fetched via `swift_toolchain`.
- **🖥️ Standalone Executables**: Multi-platform binary linking with
  `-static-stdlib` support on Linux.

---

## Quick Start

### 1. Configure `.plzconfig`

Add the Swift plugin configuration to your project's `.plzconfig`:

```ini
[Plugin "swift"]
Target = //plugins:swift
SwiftcTool = swiftc
LlvmProfdataTool = llvm-profdata
LlvmCovTool = llvm-cov

[cover]
fileextension = .swift
```

### 2. Define a Library

`src/math/BUILD`:

```python
swift_library(
    name = "math",
    srcs = ["Math.swift"],
    module_name = "Math",
    visibility = ["PUBLIC"],
)
```

`src/math/Math.swift`:

```swift
public struct Math {
    public static func add(_ a: Int, _ b: Int) -> Int {
        return a + b
    }
}
```

### 3. Write Modern Unit Tests (`swift-testing`)

`src/math/BUILD`:

```python
swift_test(
    name = "math_test",
    srcs = ["MathTests.swift"],
    deps = [":math"],
    framework = "swift-testing",
)
```

`src/math/MathTests.swift`:

```swift
import Testing
import Math

@Suite("Math Tests")
struct MathTests {
    @Test("Addition test")
    func testAdd() {
        #expect(Math.add(2, 3) == 5)
    }
}
```

Run tests:

```bash
./pleasew test //src/math:math_test
```

Run tests with code coverage:

```bash
./pleasew cover //src/math:math_test
```

### 4. Create an Executable Binary

`src/app/BUILD`:

```python
swift_binary(
    name = "app",
    srcs = ["main.swift"],
    deps = ["//src/math:math"],
)
```

`src/app/main.swift`:

```swift
import Math

let result = Math.add(10, 20)
print("Result: \(result)")
```

Run the binary:

```bash
./pleasew run //src/app:app
```

---

## Documentation Index

- **[Rule Reference](rules.md)**: Detailed API reference for `swift_library`,
  `swift_binary`, `swift_test`, and `swift_toolchain`.
- **[Architecture & Design](architecture.md)**: Compilation pipeline, metadata
  exchange, test runner synthesis, and coverage mechanics.
- **[Usage Guide](usage.md)**: In-depth usage guide covering testing,
  multi-module patterns, and hermetic toolchains.
