# Changelog

All notable changes to the Swift rules plugin will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.3] - 2026-09-20

### Added

- Support for internal Swift Testing JSON Event Streaming (`eventStreamOutputPath`) parsed directly in Go (`parseEventStream`).
- Detailed failure descriptions and issue records extracted directly from structured event stream payloads into JUnit XML test reports.

### Changed

- Transitioned primary test result extraction from text/xUnit output to internal NDJSON event stream processing while keeping intermediate JSON files private to temporary run directories.

## [0.1.2] - 2026-09-20

### Added

- Native JUnit/xUnit XML test report generation for `swift-testing` via `Testing.__CommandLineArguments_v0.xunitOutput`.
- Support for skipped test reporting and detection of test suite names.

### Fixed

- Correct quantity of tests reported for Swift tests with spaces, backticks, or custom display names in `swift-testing`.
- Robust fallback test output parsing for quoted test names, skipped tests, and execution status.

## [0.1.1] - 2026-09-16

### Added

- Root `build_defs/BUILD` filegroup forwarding `:swift` and `:constants` targets
  matching the Kotlin plugin structure.
- Automatically compile `swift_library` modules with `-enable-testing` by
  default unless `-c opt` is passed.

### Changed

- Standardized `please_swift` command usage and flag descriptions to use
  `sources` across all subcommands (`compile`, `binary`, and `testrunner`).

### Fixed

- Fixed Please f-string quote lexing error by using `flags_joined` across
  `swift_library`, `swift_binary`, and `swift_test`.
- Updated `plugins/BUILD` to target `//build_defs/swift:swift` explicitly.

## [0.1.0] - 2026-09-14

### Added

- First-class hermetic Swift build definitions:
  - `swift_library`: Compiles Swift sources into static libraries (`.a`), Swift
    module interfaces (`.swiftmodule`), documentation indexes (`.swiftdoc`), and
    dependency metadata manifests (`swift_metadata.json`).
  - `swift_binary`: Standalone executable compilation with entry point detection
    and `-static-stdlib` support on Linux.
  - `swift_test`: Sandboxed unit and integration testing powered by
    `swift-testing` (`import Testing`, `@Test`, `#expect(...)`) with automatic
    async test runner synthesis, JUnit XML reporting (`test.results`), and
    Cobertura line coverage (`test.coverage`).
  - `swift_toolchain`: Hermetic toolchain configuration for assembling official
    Swift 6.3.3 release packages across Linux and macOS.
- `tools/please_swift`: Pure Go orchestrator binary with subcommands for
  `compile`, `binary`, and `testrunner` (coverage normalization, test event
  parsing, and entrypoint synthesis).
- Full documentation suite under `docs/swift/` including overview, architecture,
  rules reference, and usage guide.
