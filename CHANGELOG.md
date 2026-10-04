# Changelog

All notable changes to the Please Kotlin rules plugin will be documented in this
file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- `kotlin_test` now reads the test results from every JUnit report the launcher
  writes. It used to copy only the first `.xml` file, so the results of the
  other test engines (a JUnit 4 class run by the vintage engine, for example)
  were lost. Errors (unexpected exceptions) and skipped tests are reported as
  such, and the counts come from the test cases.
- When no report was written (the JVM crashed, no test was found) the runner
  reports a single failing test case with the process output, instead of parsing
  that output with a regular expression. A process that fails while no test case
  does gets a failing case of its own.
- `kotlin_wasm_binary` / `kotlin_wasm_library` read the WIT file with the shared
  parser instead of line-based regular expressions. Comments, a function
  declared over several lines, a versioned package (`package a:b@1.0;`) and
  `use`, `variant` or `flags` declarations no longer confuse it, and the methods
  of a `resource` are no longer generated as functions of the enclosing
  interface. An interface in a file without a `package` line takes the package
  declared in another file of the directory. A WIT type that cannot be mapped to
  Kotlin (a `tuple`) is now an error naming the function and parameter, instead
  of a generated name that does not compile.

## [0.4.0] - 2026-10-03

### Added

- `kotlin_test` keeps the JaCoCo report converted to lcov as a declared test
  output, `plz-out/bin/<package>/<name>.lcov`, with repository-relative paths
  and the function (`FN`/`FNDA`) and branch (`BRDA`) records that Please's
  merged `coverage.xml` and `coverage.json` do not carry. It is written under
  `plz cover` and is empty otherwise. JaCoCo has no hit counts and reports
  branches as per-line counts, so counts are 0 or 1 and branch arms are
  anonymous.
- `please_kotlin testrunner --lcov-file` flag.
- Branchy coverage fixture (`test/kotlin/branches:branches_test`) and a CI smoke
  test asserting the exported functions and branches.

### Fixed

- `plz cover` reported 0% for `kt_test` targets because JaCoCo failed to analyze
  third-party classes bundled into the test jar. Each jar now records its own
  project classes and only those are reported; the test target's own sources are
  excluded.
- Coverage processing errors are now printed as warnings instead of being
  silently discarded.

### Documentation

- Documented the raw lcov export: location, contents, JaCoCo's limits, lifetime.
- Documented the required `[cover]` setting, report location, format and
  contents for Kotlin.
- Added a CI coverage smoke test.

## [0.3.5] - 2026-09-23

### Added

- `kt_wasm_library` and `kotlin_wasm_library` build rules producing reusable
  WebAssembly `.klib` libraries.
- Modular library compilation mode in `please_kotlin_wasm` emitting `.klib`
  artifacts directly without intermediate Wasm linking.
- Automatic discovery of transitive `.klib` dependencies staged in the build
  sandbox by Please via `needs_transitive_deps = True`.
- Integration test suite and verification scripts for Wasm library consumers.

## [0.3.4] - 2026-09-22

### Fixed

- Support whitespace-separated and comma-separated sources and dependencies in
  `please_kotlin_wasm`, allowing targets with multiple source files to compile
  correctly.
- Hardened bash parameter expansion in `build_defs/kotlin/kotlin.build_defs`
  against `set -u` unbound variable checks when expanding optional `SRCS` and
  `DEPS` environment variables.

## [0.3.3] - 2026-09-22

### Added

- Native `wasm-wasi` WebAssembly target support for `kt_wasm_binary` and
  `kotlin_wasm_binary`, enabling standalone execution under standard WASI
  runtimes (such as Wasmtime) without Node.js or JavaScript shims.
- Hermetic bundling of `kotlin-stdlib-wasm-wasi.klib` (v2.1.10) in
  `kotlin_toolchain` with automatic fallback resolution in the
  `please_kotlin_wasm` orchestrator.
- End-to-end integration test `//test/kotlin/wasm:disjoint_set_wasi` validating
  WASI module compilation and execution.

## [0.3.2] - 2026-09-22

### Changed

- Separate package generation: WIT interface is emitted into its declared WIT
  package (`iface.Package`), and bridge is emitted into `targetPkg` importing
  `iface.Package.*`.
- Auto-detect class implementation: automatically discovers class in sources
  matching WIT interface name when `impl` option is not explicitly specified.
- Enforce strict hermeticity: removed non-hermetic Homebrew fallback candidate
  path from Wasm stdlib resolution.

## [0.3.1] - 2026-09-20

### Added

- Precompiled release binaries for `please_kotlin_wasm` distributed via
  `remote_file` in `tools/BUILD`.
- Hermetic Wasm toolchain execution without requiring host Go or `go-rules`.

## [0.3.0] - 2026-09-20

### Added

- Support WebAssembly compilation via `kt_wasm_binary` (and
  `kotlin_wasm_binary`) build rules targeting `wasm-wasi` and `wasm-js`.
- Dedicated `please_kotlin_wasm` compiler orchestrator handling multi-phase
  compilation, Wasm klib linking, and artifact generation.
- Automated WIT bridge generation in `please_kotlin_wasm` producing
  `@WasmExport` trampolines and typed interfaces from WIT definitions.
- Integration tests and comprehensive guides for Kotlin WebAssembly architecture
  and multi-language interop.

## [0.2.0] - 2026-09-19

### Added

- Standalone `//tools/kotlin_toolchain:toolchain` package providing public
  hermetic toolchain targets (`kotlinc`, `java`, `jacoco-agent`, `jacoco-cli`).
- Configured `.plzconfig` to use `//tools/kotlin_toolchain:toolchain` by
  default.

### Changed

- Refactored `test/kotlin/toolchain/BUILD` to reference the shared toolchain
  package instead of instantiating an inline toolchain.
- Scoped CI build step to include `//tools/kotlin_toolchain/...`.

## [0.1.3] - 2026-09-13

### Added

- Distribute precompiled hermetic `please_kotlin` binaries via `tools/BUILD`
  `remote_file`.
- Configure default `PleaseKotlinTool` to `//tools:please_kotlin`.

## [0.1.2] - 2026-09-13

### Fixed

- Expose `///kotlin//build_defs:kotlin` entry point via `build_defs/BUILD`
  filegroup.

## [0.1.1] - 2026-09-13

### Fixed

- Eliminate Starlark parsing error by removing unsupported `break` keywords in
  `_derive_main_class` and `_derive_test_class`.
- Format `CHANGELOG.md` to conform to repository Prettier standards.
- Integrate `kotlin` branch into CI workflow triggers and automated downstream
  sync.

## [0.1.0] - 2026-09-13

### Added

- Initial release of Please Kotlin rules (`kotlin_library`, `kotlin_binary`,
  `kotlin_test`, `kt_library`, `kt_binary`, `kt_test`, `kt_test_suite`,
  `kotlin_jvm_import`, `maven_jar`):
  - 100% hermetic `kotlin_toolchain` rule assembling `kotlinc` 2.1.10, Temurin
    OpenJDK 21, and JaCoCo 0.8.12.
  - Multi-platform precomputed SHA-256 checksums across Linux (`x86_64`,
    `aarch64`) and macOS (`aarch64`, `x86_64`).
  - Native helper tool `please_kotlin` in Go orchestrating compilation, JAR
    packaging, and JUnit 5 test execution.
  - First-class JUnit 5 Platform integration via ConsoleLauncher with mandatory
    `test_class` and user-managed test libraries via `maven_jar`.
  - Automatic `main_class` derivation for executable application binaries.
  - Built-in `maven_jar` rule supporting coordinate resolution
    (`id = "group:artifact:version"`) and optional SHA-256 integrity checks.
  - Full code coverage support (`plz cover`) integrating JaCoCo agent
    instrumentation, report parsing, and Please-compatible GCOV/LCOV generation.
  - Automated `test.results` JUnit XML generation compatible with Please and CI
    test reporting.
  - Comprehensive integration tests in `test/kotlin/` verifying hermetic
    compilation, execution, JUnit 5 testing, and coverage.
