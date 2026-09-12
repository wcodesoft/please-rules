# Please Rules Documentation

Welcome to the documentation portal for **`please-rules`**, a collection of
extensible, hermetic language plugins and build rules for the
[Please](https://please.build) build system.

---

## Getting Started

- **[Integration & Getting Started](getting-started/README.md)**: How to
  declare, configure, and consume rulesets from this repository in your Please
  projects.

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

```txt
please-rules/
├── build_defs/          # Starlark rule definitions by language (e.g. rust/)
├── tools/               # Native compiler orchestrators and helper CLI binaries
├── plugins/             # Plugin repository declarations exposed to consumers
├── docs/                # Documentation portal and language-specific guides
│   ├── getting-started/ # Project integration and onboarding guides
│   └── rust/            # Rust architecture, rule reference, and guides
└── test/                # End-to-end integration test suites
```

---

## Tooling & Code Quality

- **[Veritas Code Quality & Telemetry](veritas.md)**: Local static analysis and
  automated test telemetry configuration.
