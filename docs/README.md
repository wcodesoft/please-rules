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
  ### Swift (`///swift`)

Hermetic, performant build rules for Swift 6+ with native `swift-testing`
support, multi-module linking, and code coverage on the
[`swift`](https://github.com/wcodesoft/please-rules/tree/swift) branch:

- **[Swift Documentation Overview](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/README.md)**:
  Entry point and quick overview.
- **[Rule Reference](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/rules.md)**:
  Detailed API reference for `swift_library`, `swift_binary`, `swift_test`, and
  `swift_toolchain`.
- **[Architecture & Design](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/architecture.md)**:
  Under-the-hood design of `please_swift`, compiler orchestration, test runner
  synthesis, and coverage.
- **[Usage Guide](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/usage.md)**:
  Multi-module project layouts, `swift-testing` suites, and coverage workflows.

### Kotlin (`///kotlin`)

100% hermetic support for compiling and testing Kotlin applications with OpenJDK
21, JUnit XML telemetry, and JaCoCo code coverage on the
[`kotlin`](https://github.com/wcodesoft/please-rules/tree/kotlin) branch:

- **[Kotlin Documentation Overview](https://github.com/wcodesoft/please-rules/blob/kotlin/docs/kotlin/README.md)**:
  Entry point and feature summary.
- **[Rule Reference](https://github.com/wcodesoft/please-rules/blob/kotlin/docs/kotlin/rules.md)**:
  API reference for `kotlin_library`, `kotlin_binary`, `kotlin_test`, and
  `kotlin_jvm_import`.
- **[Architecture & Design](https://github.com/wcodesoft/please-rules/blob/kotlin/docs/kotlin/architecture.md)**:
  Sysroot layout, `please_kotlin` CLI orchestrator, and JaCoCo coverage
  pipeline.
- **[Usage Guide](https://github.com/wcodesoft/please-rules/blob/kotlin/docs/kotlin/usage.md)**:
  `.plzconfig` setup, third-party jar dependencies, and testrunner execution.

### Swift (`///swift`)

Hermetic, performant build rules for Swift 6+ with native `swift-testing`
support, multi-module linking, and code coverage on the
[`swift`](https://github.com/wcodesoft/please-rules/tree/swift) branch:

- **[Swift Documentation Overview](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/README.md)**:
  Entry point and quick overview.
- **[Rule Reference](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/rules.md)**:
  Detailed API reference for `swift_library`, `swift_binary`, `swift_test`, and
  `swift_toolchain`.
- **[Architecture & Design](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/architecture.md)**:
  Under-the-hood design of `please_swift`, compiler orchestration, test runner
  synthesis, and coverage.
- **[Usage Guide](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/usage.md)**:
  Multi-module project layouts, `swift-testing` suites, and coverage workflows.

### TypeScript & Deno (`///ts`)

Hermetic TypeScript and Deno build rules with zero `node_modules` and offline
ephemeral import maps on the
[`ts`](https://github.com/wcodesoft/please-rules/tree/ts) branch:

- **[TypeScript Documentation Overview](https://github.com/wcodesoft/please-rules/blob/ts/docs/ts/README.md)**:
  Entry point and quick overview.
- **[Rule Reference](https://github.com/wcodesoft/please-rules/blob/ts/docs/ts/rules.md)**:
  API reference for `ts_library`, `ts_binary`, `ts_bundle`, `ts_test`, and
  `ts_module`.
- **[Architecture & Design](https://github.com/wcodesoft/please-rules/blob/ts/docs/ts/architecture.md)**:
  Sandboxing, ephemeral import maps, and `please_ts` orchestrator.
- **[Usage Guide](https://github.com/wcodesoft/please-rules/blob/ts/docs/ts/usage.md)**:
  Testing patterns, third-party npm tarballs, and web bundling.

### WebAssembly Interface Types (`///wit`)

Hermetic build rules for WebAssembly Interface Types (WIT) bundling, multi-world
discovery, and multi-language binding generation across Swift, Kotlin, Rust, Go,
TypeScript, C++, and Python on the
[`wit`](https://github.com/wcodesoft/please-rules/tree/wit) branch:

- **[WIT Branch & Rules](https://github.com/wcodesoft/please-rules/tree/wit)**:
  Entry point and rule definitions.
- **Rule Reference**:
  - `wit_library`: Packages one or more `.wit` interface definition files.
  - `_wit_bindgen`: Centralized private binding generator engine.
  - Language wrappers: `swift_wit_bindgen`, `kt_wit_bindgen`,
    `rust_wit_bindgen`, `go_wit_bindgen`, `ts_wit_bindgen`, `cc_wit_bindgen`,
    `python_wit_bindgen` (`py_wit_bindgen`).
  - Unified `worlds = [...]` list parameter supporting full-package or targeted
    world generation.
- **Toolchain & CLI**: Hermetic `wit_toolchain` downloading `wit-bindgen`
  (v0.62.0) and `wasm-tools` (v1.259.0), driven by the `please_wit` Go
  orchestrator CLI.

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
