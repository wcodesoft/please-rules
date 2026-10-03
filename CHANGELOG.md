# Changelog

All notable changes to the Swift rules plugin will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-10-03

### Added

- `swift_test` keeps the raw lcov report as a declared test output,
  `plz-out/bin/<package>/<name>.lcov`, with repository-relative paths, function
  records (`FN`/`FNDA`, Swift-mangled names) and, for lines that ran only
  partly, synthesized branch records (`BRDA`) derived from `llvm-cov`'s sub-line
  regions: Swift emits no branch regions, so this covers cases such as the
  right-hand side of `a && b` or the body of an untaken `else if` that shares a
  line with other code. It is written under `plz cover` and is empty otherwise.
- `please_swift testrunner --lcov-file` flag.
- Branchy coverage fixture (`test/swift/branches:branches_test`) and a CI smoke
  test asserting the exported functions and branches.

### Changed

- `swift_test` is now a binary target like the other languages' test rules, so
  its outputs are under `plz-out/bin` instead of `plz-out/gen`.
- Removed a leftover debug print from the test runner and the stray
  `fileextension = .go` from the repository `[cover]` config.

### Fixed

- `plz cover` failed to link the instrumented test binary because the test
  runner scanned the whole test directory and linked every archive found in the
  Swift toolchain. It now uses the link inputs saved by the build phase.

- Tests and binaries that depend on a library which itself depends on another
  Swift library failed to link on Linux (`undefined reference`): archives are
  now ordered so each library precedes its dependencies, using the module
  dependencies recorded in `swift_metadata.json`.
- `plz cover` now also works for those transitive dependency chains: the build
  phase saves the transitive link inputs in a `<test>.deps` output for the
  instrumented rebuild.

### Documentation

- Documented the required `[cover]` setting, report location, format and
  contents for Swift, and the raw lcov export.
- Added CI coverage smoke tests.

## [0.2.0] - 2026-09-27

### Added

- Support for minimal Swift toolchain distributions hosted on GitHub Releases,
  stripping non-compiler assets (LLDB, SPM, DocC, SourceKit-LSP, and Python) to
  reduce download size from ~1GB to ~150–180MB.
- `scripts/strip_swift_toolchain.py` automated stripping utility for official
  Swift release archives.
- Automated `.github/workflows/build-swift-minimal.yml` matrix workflow to build
  and distribute minimal toolchains.
- Updated `SWIFT_PLATFORM_MAP` in `build_defs/swift/constants.build_defs` to
  default to minimal toolchain releases for Swift 6.3.3.

## [0.1.3] - 2026-09-20

### Added

- Support for internal Swift Testing JSON Event Streaming
  (`eventStreamOutputPath`) parsed directly in Go (`parseEventStream`).
- Detailed failure descriptions and issue records extracted directly from
  structured event stream payloads into JUnit XML test reports.

### Changed

- Transitioned primary test result extraction from text/xUnit output to internal
  NDJSON event stream processing while keeping intermediate JSON files private
  to temporary run directories.

## [0.1.2] - 2026-09-20

### Added

- Native JUnit/xUnit XML test report generation for `swift-testing` via
  `Testing.__CommandLineArguments_v0.xunitOutput`.
- Support for skipped test reporting and detection of test suite names.

### Fixed

- Correct quantity of tests reported for Swift tests with spaces, backticks, or
  custom display names in `swift-testing`.
- Robust fallback test output parsing for quoted test names, skipped tests, and
  execution status.

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
