# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-09-13

### Added

- Multi-language Hub-and-Spoke Gitflow architecture and AI guidelines:
  - Added multi-language architecture guide in `docs/gitflow.md`.
  - Added `.agents/skills/multi-language-gitflow/SKILL.md` skill for automated
    Gitflow compliance.
  - Added `AGENTS.md` root workspace rules enforcing one-way merging, rebase
    prohibition, and language branch isolation.
- Created dedicated `rust` branch for language-specific releases and
  development.

### Changed

- Reorganized the test suite to support multi-language coexistence:
  - Moved integration tests from `test/bin`, `test/lib`, and `test/toolchain`
    into namespaced directory `test/rust/` (`test/rust/bin`, `test/rust/lib`,
    `test/rust/toolchain`).
  - Updated toolchain and test references in `.plzconfig` and internal `BUILD`
    files.
- Made build definition subincludes portable
  (`subinclude("//build_defs/rust:constants")`) to avoid coupling to external
  subrepo aliases.

## [0.3.3] - 2026-09-13

### Added

- Code coverage support for Rust rules (`plz cover`)
  ([#5](https://github.com/wcodesoft/please-rules/issues/5)):
  - Added `-C instrument-coverage` instrumentation support to `rust_library`,
    `rust_bin`, and `rust_test`.
  - Added `llvm-tools` archive downloading and SHA-256 verification in
    `rust_toolchain` with `:toolchain|llvm-profdata` and `:toolchain|llvm-cov`
    entry points for Rust 1.85.0 across Linux (`x86_64`, `aarch64`) and macOS
    (`x86_64`, `aarch64`).
  - Added host toolchain discovery for `llvm-profdata` and `llvm-cov` in
    `tools/please_rust/toolchain` (searching `PATH`, `/usr/lib/llvm-*/bin`,
    rustup toolchains, and `rustc --print sysroot`).
  - Extended `testrunner` to set `LLVM_PROFILE_FILE`, merge raw profiles using
    `llvm-profdata`, and export coverage using `llvm-cov`.
  - Added LCOV path normalization and GCOV format generation for native
    compatibility with Please's coverage engine, supporting summary and
    line-by-line (`plz cover -l`) output.
  - Added `[cover] fileextension = .rs` and plugin configuration options
    `Coverage`, `LlvmProfdataTool`, and `LlvmCovTool` in `.plzconfig`.
  - Documented coverage configuration and usage in `docs/rust/usage.md`.
- Added GitHub Actions CI workflow (`.github/workflows/ci.yml`) for Prettier
  markdown checks, Go code formatting (`gofmt`), and Please build & test.
- Documented robust git hooks, nohup detachment, and hook logging.

### Changed

- Refactored `tools/please_rust` (`testrunner`, `compile`, `download`) into
  modular files to lower cyclomatic and cognitive complexity and improve test
  coverage.
- Bound `[Plugin "rust"]` to hermetic `//build_defs/rust:toolchain` in
  repository `.plzconfig`.

## [0.3.2] - 2026-09-12

### Fixed

- Fixed plugin config resolution in `rust.build_defs`: use
  `CONFIG.RUST.RUSTC_TOOL` directly instead of `CONFIG.get("RUST.RUSTC_TOOL")`,
  ensuring hermetic `RustcTool` is properly passed to all rules as
  `--rustc "$TOOLS_RUSTC"`.

### Removed

- Removed legacy `fetch` subcommand and package (`tools/please_rust/fetch`),
  making `please_rust` completely Cargo-free.
- Removed unused `FindCargo` helper from `tools/please_rust/toolchain` and
  `CargoTool` configuration from `.plzconfig`.

## [0.3.1] - 2026-09-12

### Added

- Hermetic Rust Toolchain (`rust_toolchain`)
  ([#3](https://github.com/wcodesoft/please-rules/issues/3)):
  - Added `rust_toolchain` rule to download standalone `rustc` and `rust-std`
    distributions from `static.rust-lang.org`.
  - Built-in cryptographic SHA-256 verification with pre-populated hashes for
    Rust 1.85.0 on Linux (`x86_64`, `aarch64`) and macOS (`x86_64`, `aarch64`).
  - Assembles a self-contained compiler sysroot into `$OUT` with
    `:toolchain|rustc` entry point.
  - Seamless integration with `rust_library`, `rust_bin`, `rust_test`, and
    `rust_crate` via `CONFIG.RUST.RUSTC_TOOL` (configured globally in
    `.plzconfig`), keeping target rule signatures completely clean.
  - Eliminates the need for host-installed toolchains (`rustup`) and environment
    variable pass-through (`passenv = PATH, HOME`).
  - Added absolute path resolution in `tools/please_rust/toolchain` for tool
    overrides.
  - Added integration test suite (`test/toolchain`) verifying hermetic
    compilation and testing.

## [0.3.0] - 2026-09-11

### Added

- Hermetic crate download and verification:
  - Added `download` subcommand in `please_rust` to fetch `.crate` archives
    directly from crates.io with mandatory SHA-256 verification and automatic
    extraction of `edition` and `lib.rs` path into `crate_meta.json`.
  - Added `hash` subcommand in `please_rust` to query and print the SHA-256
    digest of any crate from crates.io.
- Headless C compilation for native extensions:
  - Added `compile-c` subcommand in `please_rust` and `compilec` Go package to
    compile `.c` sources into static archives (`.a`) using the system C compiler
    (`cc`/`gcc`/`clang`), enabling native grammar crates (e.g. `tree-sitter-*`)
    without Cargo `build.rs`.
- Enhanced compiler flags and features:
  - Added `--native-lib` flag to `please_rust compile` for linking `.a` static
    archives via `rustc -L native=<dir> -l static=<lib>`.
  - Added `--features` support to `please_rust compile` translating to
    `--cfg feature="<name>"`.
  - Added `flags` parameter to `rust_crate`.
  - Added `env` dictionary parameter to `rust_library`, `rust_bin`, and
    `rust_test` to export compile-time environment variables (such as
    `VERITAS_VERSION`).
  - Added automatic `--extern proc_macro` flag when compiling procedural macro
    crates (`proc-macro`).
  - Populated `CARGO_PKG_VERSION_MAJOR`, `CARGO_PKG_VERSION_MINOR`, and
    `CARGO_PKG_VERSION_PATCH` environment variables during `rustc` invocations.
- Test runner integration:
  - Added `test_tools` propagation to `rust_test` for hermetic `$TOOL` discovery
    during test execution.
  - Set explicit `--pkg` target naming for `rust_test` execution.

### Changed

- Re-architected `rust_crate` to compile third-party crates hermetically and
  directly with `rustc`, eliminating dependency on Cargo at build time and
  requiring explicit `sha256` integrity hashes.

### Removed

- Removed legacy embedded Python test-runner generator script from `compile.go`
  in favor of the pure-Go `please_rust test-runner` subcommand.

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
