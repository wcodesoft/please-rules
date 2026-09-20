# Changelog

All notable changes to the Please Kotlin rules plugin will be documented in this
file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.1] - 2026-09-20

### Added

- Precompiled release binaries for `please_kotlin_wasm` distributed via `remote_file` in `tools/BUILD`.
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
