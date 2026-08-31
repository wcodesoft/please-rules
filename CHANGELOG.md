# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-08-31

### Added

- Added `proc_macro` support to `rust_crate` for compiling procedural macro
  shared libraries (`.so`).
- Added `meta = True` support to `rust_crate` for aggregator/virtual crates.
- Added metadata parameters (`crate_name`, `pkg_name`, `license`, `repository`)
  to `rust_crate`.
- Enabled transitive dependency propagation (`needs_transitive_deps = True`) on
  all Rust build rules (`rust_library`, `rust_bin`, `rust_test`, `rust_crate`).

## [0.1.0] - 2026-08-31

### Added

- Initial release of Please Rust build rules (`please-rules`).
- `rust_library`: Rule for building Rust library crates into `.rlib` archives.
- `rust_bin`: Rule for compiling executable Rust binaries.
- `rust_test`: Rule for running Rust test suites with automatic test discovery
  and JUnit XML test reporting.
- `rust_crate`: Rule for downloading and compiling third-party dependencies from
  `crates.io` using Cargo.
- `please_rust`: Hermetic Go helper tool implementing compilation orchestration
  (`compile`), crate fetching (`fetch`), and test output conversion
  (`test-runner`).
- Flexible source entrypoint resolution supporting flat structures (`lib.rs`,
  `main.rs`) and nested directory structures (`src/lib.rs`, `src/main.rs`).
- VSCode workspace configuration (`.vscode/settings.json`) and Prettier
  formatting (`.prettierrc`).
- Comprehensive documentation and developer guide in `docs/` and `README.md`.
