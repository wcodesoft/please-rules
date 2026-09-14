# Swift Build Rules Reference

This document provides a comprehensive API reference for all build rules
exported by `///swift`.

---

## `swift_library`

Compiles a collection of Swift source files into a static library archive
(`.a`), Swift module interface (`.swiftmodule`), documentation index
(`.swiftdoc`), and metadata manifest (`swift_metadata.json`).

```python
swift_library(
    name: str,
    srcs: list,
    deps: list = None,
    module_name: str = "",
    swift_version: str = "",
    flags: list = None,
    static_stdlib: bool = False,
    visibility: list = None,
    labels: list = None,
)
```

### Arguments

| Argument        | Type        | Default            | Description                                                                                                    |
| --------------- | ----------- | ------------------ | -------------------------------------------------------------------------------------------------------------- |
| `name`          | `str`       | Required           | Unique target name. If `module_name` is omitted, `name` is used as the Swift module name.                      |
| `srcs`          | `list[str]` | Required           | List of Swift source files (`.swift`).                                                                         |
| `deps`          | `list[str]` | `[]`               | Dependent `swift_library` targets. Transitive module directories and archive paths are automatically provided. |
| `module_name`   | `str`       | `""`               | The module name emitted and consumed by `import <module_name>`. Defaults to `name`.                            |
| `swift_version` | `str`       | `""`               | Swift language version compatibility mode (e.g. `"5"`, `"6"`). Passed to `-swift-version`.                     |
| `flags`         | `list[str]` | `[]`               | Additional flags forwarded directly to `swiftc` during compilation.                                            |
| `static_stdlib` | `bool`      | `False`            | Whether to link the Swift standard library statically (Linux only).                                            |
| `visibility`    | `list[str]` | `None`             | Visibility list (e.g. `["PUBLIC"]`).                                                                           |
| `labels`        | `list[str]` | `["swift", "lib"]` | Build rule labels.                                                                                             |

---

## `swift_binary`

Compiles Swift source files and links them with dependencies to produce a
runnable standalone executable.

```python
swift_binary(
    name: str,
    main: str = "",
    srcs: list = None,
    deps: list = None,
    module_name: str = "",
    flags: list = None,
    static_stdlib: bool = False,
    visibility: list = None,
    labels: list = None,
)
```

### Arguments

| Argument        | Type        | Default            | Description                                                                                                               |
| --------------- | ----------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------- |
| `name`          | `str`       | Required           | Output executable name.                                                                                                   |
| `main`          | `str`       | `""`               | Explicit main source file containing top-level code or `@main` attribute. If specified, automatically included in `srcs`. |
| `srcs`          | `list[str]` | `[]`               | Swift source files for the binary.                                                                                        |
| `deps`          | `list[str]` | `[]`               | Dependent `swift_library` targets. Transitive archives (`.a`) are linked in reverse topological order.                    |
| `module_name`   | `str`       | `""`               | Module name for compilation. Defaults to `name`.                                                                          |
| `flags`         | `list[str]` | `[]`               | Additional flags forwarded to `swiftc` during linking.                                                                    |
| `static_stdlib` | `bool`      | `False`            | If `True`, passes `-static-stdlib` to `swiftc` on Linux for self-contained executables.                                   |
| `visibility`    | `list[str]` | `None`             | Visibility list.                                                                                                          |
| `labels`        | `list[str]` | `["swift", "bin"]` | Build rule labels.                                                                                                        |

---

## `swift_test`

Compiles and executes a Swift test target using `swift-testing` or legacy
`XCTest`. Produces standardized JUnit XML test reports and Cobertura line
coverage XML.

```python
swift_test(
    name: str,
    srcs: list,
    deps: list = None,
    data: list = None,
    framework: str = "swift-testing",
    flags: list = None,
    visibility: list = None,
    labels: list = None,
)
```

### Arguments

| Argument     | Type        | Default             | Description                                                                   |
| ------------ | ----------- | ------------------- | ----------------------------------------------------------------------------- |
| `name`       | `str`       | Required            | Unique test target name.                                                      |
| `srcs`       | `list[str]` | Required            | Swift test source files (containing `@Test` or `XCTestCase`).                 |
| `deps`       | `list[str]` | `[]`                | Libraries under test and required test dependencies.                          |
| `data`       | `list[str]` | `[]`                | Runtime data files accessible during test execution.                          |
| `framework`  | `str`       | `"swift-testing"`   | Test framework to use. Options are `"swift-testing"` (default) or `"xctest"`. |
| `flags`      | `list[str]` | `[]`                | Additional flags forwarded to `swiftc`.                                       |
| `visibility` | `list[str]` | `None`              | Visibility list.                                                              |
| `labels`     | `list[str]` | `["swift", "test"]` | Build rule labels.                                                            |

### Test Execution Details

- **Test Runner Synthesis**: For `swift-testing`, if no source file declares
  `@main`, the test orchestrator automatically synthesizes an async entry point
  invoking `Testing.__swiftPMEntryPoint()` and compiles with
  `-parse-as-library -lTesting`.
- **JUnit Reporting**: Captures test outcomes (pass, fail, skip) and generates
  standard JUnit XML in `test.results`.
- **Code Coverage**: When executed via `pleasew cover`, the orchestrator merges
  LLVM execution profiles via `llvm-profdata` and exports line-level Cobertura
  coverage to `test.coverage`.

---

## `swift_toolchain`

Downloads and unpacks an official hermetic Swift toolchain archive for the host
platform, exporting `swiftc`, `swift`, `llvm-profdata`, and `llvm-cov`.

```python
swift_toolchain(
    name: str = "toolchain",
    version: str = "6.3.3",
    swift_url: str = "",
    swift_hash: str = "",
    target_platform: str = "",
    labels: list = None,
    visibility: list = None,
)
```

### Arguments

| Argument          | Type        | Default       | Description                                                                                    |
| ----------------- | ----------- | ------------- | ---------------------------------------------------------------------------------------------- |
| `name`            | `str`       | `"toolchain"` | Target name.                                                                                   |
| `version`         | `str`       | `"6.3.3"`     | Swift release version.                                                                         |
| `swift_url`       | `str`       | `""`          | Optional direct download URL. If empty, uses the verified release URL in `SWIFT_PLATFORM_MAP`. |
| `swift_hash`      | `str`       | `""`          | Expected SHA-256 hash for integrity verification.                                              |
| `target_platform` | `str`       | `""`          | Target platform string (e.g. `linux_amd64`, `linux_arm64`, `darwin_arm64`).                    |
| `visibility`      | `list[str]` | `["PUBLIC"]`  | Target visibility.                                                                             |
