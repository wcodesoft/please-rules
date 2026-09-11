# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).
## [0.2.7] - 2026-09-11

### Added

- Hermetic crate download and verification:
  - Added `download` subcommand in `please_rust` to fetch `.crate` archives directly from crates.io with mandatory SHA-256 verification and automatic extraction of `edition` and `lib.rs` path into `crate_meta.json`.
  - Added `hash` subcommand in `please_rust` to query and print the SHA-256 digest of any crate from crates.io.
- Headless C compilation for native extensions:
  - Added `compile-c` subcommand in `please_rust` and `compilec` Go package to compile `.c` sources into static archives (`.a`) using the system C compiler (`cc`/`gcc`/`clang`), enabling native grammar crates (e.g. `tree-sitter-*`) without Cargo `build.rs`.
- Enhanced compiler flags and features:
  - Added `--native-lib` flag to `please_rust compile` for linking `.a` static archives via `rustc -L native=<dir> -l static=<lib>`.
  - Added `--features` support to `please_rust compile` translating to `--cfg feature="<name>"`.
  - Added `flags` parameter to `rust_crate`.
  - Added `env` dictionary parameter to `rust_library`, `rust_bin`, and `rust_test` to export compile-time environment variables (such as `VERITAS_VERSION`).
  - Added automatic `--extern proc_macro` flag when compiling procedural macro crates (`proc-macro`).
  - Populated `CARGO_PKG_VERSION_MAJOR`, `CARGO_PKG_VERSION_MINOR`, and `CARGO_PKG_VERSION_PATCH` environment variables during `rustc` invocations.
- Test runner integration:
  - Added `test_tools` propagation to `rust_test` for hermetic `$TOOL` discovery during test execution.
  - Set explicit `--pkg` target naming for `rust_test` execution.

### Changed

- Re-architected `rust_crate` to compile third-party crates hermetically and directly with `rustc`, eliminating dependency on Cargo at build time and requiring explicit `sha256` integrity hashes.

### Removed

- Removed legacy embedded Python test-runner generator script from `compile.go` in favor of the pure-Go `please_rust test-runner` subcommand.

## [0.2.6] - 2026-09-10

### Fixed

- Set default Rust edition to `2024` across all build definitions.

## [0.2.5] - 2026-09-10

### Changed

- Standardized entrypoint specification exclusively on `main` across
  `rust_library`, `rust_bin`, and `rust_test`, removing redundant `main_src`
  parameter.
- Updated documentation and rule references to reflect the simplified rule
  signatures.

## [0.2.4] - 2026-09-10

### Fixed

- Prioritized `mod.rs` and `lib.rs` when auto-detecting crate entrypoint files
  from `srcs`.

## [0.2.3] - 2026-09-10

### Changed

- Comprehensive static code quality refactoring across `tools/please_rust`:
  - Decomposed `resolveMainSrc`, `BuildRustcArgs`, `Run`, `FetchCrate`,
    `FetchAll`, `getCargoEnv`, and CLI dispatching.
  - Decreased maximum cognitive complexity from 39 to 13, and maximum cyclomatic
    complexity from 23 to 10.
  - Eliminated code duplication down to 2.0% across all Go utilities.
- Simplified default configuration values in `build_defs/rust/rust.build_defs`
  to ensure compatibility across all Please Starlark environments.

## [0.2.2] - 2026-09-10

### Fixed

- Fixed Starlark configuration attribute resolution using
  `hasattr(CONFIG, "RUST")` for robust defaults handling.

## [0.2.1] - 2026-09-10

### Added

- Added `main` parameter support to `rust_library`, `rust_bin`, and `rust_test`
  rules for explicit root entrypoint specification.
- Integrated Veritas static code analysis and automated test telemetry via
  `pleasew` test hook and Git post-commit quality upload.

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
