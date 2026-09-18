# Changelog

All notable changes to the TypeScript / Deno rules plugin will be documented in
this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.2] - 2026-09-18

### Fixed

- Verified and updated precompiled binary checksums in `tools/BUILD` for
  `please_ts` across `linux_amd64`, `linux_arm64`, `darwin_amd64`, and
  `darwin_arm64`.

## [0.1.1] - 2026-09-17

### Fixed

- Fixed Starlark f-string quote syntax error when passing CLI `flags` in
  `ts_library`, `ts_binary`, `ts_bundle`, `ts_test`, and `ts_browser_test`.
- Standardized CLI flag description and usage from singular `source files` to
  `sources` across `please_ts` subcommands.

## [0.1.0] - 2026-09-13

### Added

- First-class hermetic TypeScript and JavaScript build definitions:
  - `ts_module`: Downloads and extracts npm packages with optional integrity
    verification.
  - `ts_library`: Hermetic type checking via `deno check` and ephemeral import
    maps.
  - `ts_binary`: Standalone self-contained native executable compilation via
    `deno compile`.
  - `ts_bundle`: Web/browser distribution asset bundling.
  - `ts_test`: Sandboxed unit and DOM tests using `deno test` or `vitest` with
    native JUnit XML and Cobertura XML test coverage.
  - `ts_browser_test`: Hermetic in-browser testing using `@vitest/browser` and
    sandboxed browser binaries.
- Hermetic standalone Deno toolchain (`ts_toolchain`) for `linux_amd64`,
  `linux_arm64`, `darwin_amd64`, and `darwin_arm64`.
- `tools/please_ts` orchestrator tool written in Go with commands for `compile`,
  `bundle`, `testrunner`, `unpack`, and `importmap`.
