# Please Rules Documentation

Welcome to the documentation portal for **`please-rules`**, a collection of
extensible, hermetic language plugins and build rules for the
[Please](https://please.build) build system.

---

## Language Rulesets

### Rust (`///rust`)

First-class, hermetic support for compiling, testing, and managing Rust crates
and dependencies:

- **[Rust Documentation Overview](rust/README.md)**: Entry point and quick
  overview.
- **[Rule Reference](rust/rules.md)**: Detailed API reference for
  `rust_library`, `rust_bin`, `rust_test`, and `rust_crate`.
- **[Architecture & Design](rust/architecture.md)**: Under-the-hood design of
  the `please_rust` multi-subcommand binary, compilation workflows, and
  diagrams.
- **[Usage Guide](rust/usage.md)**: Integration guides, `.plzconfig` setup, and
  multi-crate layout examples.
- **[Dependencies & Toolchains](rust/dependencies.md)**: Toolchain requirements,
  automatic path discovery, and C native dependencies.

---

## Repository Structure

- `build_defs/`: Starlark build definitions for each supported language.
- `tools/`: Multi-platform helper tools and compiler orchestrators.
- `plugins/`: Plugin repository declarations exposed to consumer projects.
- `docs/`: Language-specific documentation and general guides.
- `test/`: Integration and end-to-end test suites.
