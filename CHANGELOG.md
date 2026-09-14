# Changelog

All notable changes to the Swift rules plugin will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
