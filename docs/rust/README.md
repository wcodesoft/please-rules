# Rust Rules Documentation

Welcome to the documentation for the Please build system Rust rules in
`please-rules`.

This ruleset provides first-class, hermetic support for compiling, testing, and
managing Rust crates and third-party dependencies using
[Please](https://please.build).

---

## Table of Contents

- [Architecture & Internal Design](architecture.md)
  - Overview of how Please rules interact with `please_rust`
  - Subcommands: `compile`, `download`, `compile-c`, `hash`, `test-runner`
  - Toolchain discovery and execution model
  - Architecture and workflow diagrams
- [Rule Reference](rules.md)
  - `rust_library`: Compiles `.rlib` static libraries
  - `rust_bin`: Compiles binary executables
  - `rust_test`: Compiles and executes test executables with JUnit test
    reporting
  - `rust_crate`: Hermetically downloads and compiles third-party crates with
    SHA-256 verification
  - `rust_toolchain`: Hermetically downloads and provides a reproducible Rust
    compiler sysroot
- [Dependencies & Toolchains](dependencies.md)
  - System toolchain requirements (`rustc`, C compiler `cc`, Go)
  - Automatic toolchain resolution and path overrides
  - Managing third-party crate dependencies
- [Usage & Configuration Guide](usage.md)
  - Quickstart guide
  - `.plzconfig` configuration options
  - `subinclude` patterns
  - Multi-crate project layout examples
