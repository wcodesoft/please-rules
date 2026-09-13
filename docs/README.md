# Please Rules Documentation

Welcome to the documentation portal for **`please-rules`**, a collection of
extensible, hermetic language plugins and build rules for the
[Please](https://please.build) build system.

---

## Getting Started

- **[Integration & Getting Started](getting-started.md)**: How to declare,
  configure, and consume rulesets from this repository in your Please projects.
- **[Multi-Language Gitflow Architecture](gitflow.md)**: Branching strategy,
  per-language releases, and multi-plugin Gitflow.

---

## Language Rulesets

Each language ruleset lives on its own dedicated branch. Full documentation for
each language is maintained directly in its branch:

### Rust (`///rust`)

First-class, hermetic support for compiling, testing, and managing Rust crates
and dependencies on the
[`rust`](https://github.com/wcodesoft/please-rules/tree/rust) branch:

- **[Rust Documentation Overview](https://github.com/wcodesoft/please-rules/blob/rust/docs/rust/README.md)**:
  Entry point and quick overview.
- **[Rule Reference](https://github.com/wcodesoft/please-rules/blob/rust/docs/rust/rules.md)**:
  Detailed API reference for `rust_library`, `rust_bin`, `rust_test`, and
  `rust_crate`.
- **[Architecture & Design](https://github.com/wcodesoft/please-rules/blob/rust/docs/rust/architecture.md)**:
  Under-the-hood design of the `please_rust` multi-subcommand binary,
  compilation workflows, and diagrams.
- **[Usage Guide](https://github.com/wcodesoft/please-rules/blob/rust/docs/rust/usage.md)**:
  Integration guides, `.plzconfig` setup, and multi-crate layout examples.
- **[Dependencies & Toolchains](https://github.com/wcodesoft/please-rules/blob/rust/docs/rust/dependencies.md)**:
  Toolchain requirements, automatic path discovery, and C native dependencies.

---

## Repository Structure

```txt
please-rules/
├── main branch              # Common infrastructure, base CI, shared tools, and docs
│   ├── .agents/skills/      # AI agent skills (Gitflow automation)
│   ├── .github/workflows/   # CI workflows and branch guard
│   ├── docs/                # Documentation portal, getting-started, and gitflow guides
│   └── plugins/             # Common plugin repository declarations
└── <language> branches      # Dedicated long-lived branches (e.g. rust, kotlin)
    ├── build_defs/<lang>/   # Starlark rule definitions for <language>
    ├── tools/please_<lang>/ # Native compiler orchestrator and CLI tools
    ├── docs/<lang>/         # Language architecture, rule reference, and usage guides
    └── test/<lang>/         # End-to-end integration test suites for <language>
```

---

## Tooling & Code Quality

- **[Veritas Code Quality & Telemetry](veritas.md)**: Local static analysis and
  automated test telemetry configuration.
