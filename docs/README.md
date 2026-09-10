# Rust Rules Documentation (`please-rules`)

Welcome to the documentation for the Please build system Rust rules plugin
(`please-rules`).

This plugin provides first-class, hermetic support for compiling, testing, and
managing Rust crates and third-party dependencies using
[Please](https://please.build).

---

## Table of Contents

- [Architecture & Internal Design](architecture.md)
  - Overview of how Please rules interact with `please_rust`
  - Subcommands: `compile`, `fetch`, `test-runner`
  - Toolchain discovery and execution model
  - Mermaid architecture & flow diagrams
- [Rule Reference](rules.md)
  - `rust_library`: Compiles `.rlib` static libraries
  - `rust_bin`: Compiles binary executables
  - `rust_test`: Compiles and executes test executables
  - `rust_crate`: Fetches and builds third-party crates from `crates.io`
- [Dependencies & Toolchains](dependencies.md)
  - System toolchain requirements (`rustc`, `cargo`, Go)
  - Automatic toolchain resolution and path overrides
  - Managing third-party crate dependencies
- [Usage & Configuration Guide](usage.md)
  - Quickstart guide
  - `.plzconfig` configuration options
  - `subinclude` patterns
  - Build file examples and multi-crate project layout
