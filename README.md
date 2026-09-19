# Please Rules (`please-rules`)

A curated collection of modern, hermetic build rules and plugins for the
[Please](https://please.build) build system.

This repository provides language toolchains and build definitions designed for
hermeticity, reproducible builds, and minimal host dependencies.

---

## Getting Started

To integrate and use rules from this repository in your Please project, see the
**[Getting Started & Integration Guide](docs/getting-started.md)**.

---

## Supported Languages

`please-rules` employs a [Hub-and-Spoke Gitflow](docs/gitflow.md) where each
language ruleset is developed, tested, and released on its own dedicated
long-lived branch:

| Language                 | Branch                                                            | Latest Release                                                                            | Documentation                                                                                             | Description                                                                                                         |
| :----------------------- | :---------------------------------------------------------------- | :---------------------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------ |
| **Rust (`///rust`)**     | [`rust`](https://github.com/wcodesoft/please-rules/tree/rust)     | [`[Rust] v0.5.0`](https://github.com/wcodesoft/please-rules/releases/tag/rust-v0.5.0)     | [Rust Rules Documentation](https://github.com/wcodesoft/please-rules/blob/rust/docs/rust/README.md)       | Hermetic toolchains, crates.io dependency management, C native linking, and `llvm-cov` coverage.                    |
| **Kotlin (`///kotlin`)** | [`kotlin`](https://github.com/wcodesoft/please-rules/tree/kotlin) | [`[Kotlin] v0.2.0`](https://github.com/wcodesoft/please-rules/releases/tag/kotlin-v0.2.0) | [Kotlin Rules Documentation](https://github.com/wcodesoft/please-rules/blob/kotlin/docs/kotlin/README.md) | 100% hermetic `kotlinc` 2.1 & OpenJDK 21, JUnit XML telemetry, and JaCoCo code coverage.                            |
| **Swift (`///swift`)**   | [`swift`](https://github.com/wcodesoft/please-rules/tree/swift)   | [`[Swift] v0.1.1`](https://github.com/wcodesoft/please-rules/releases/tag/swift-v0.1.1)   | [Swift Rules Documentation](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/README.md)    | Swift 6+ rules, native `swift-testing` support, multi-module static linking, and coverage.                          |
| **TypeScript (`///ts`)** | [`ts`](https://github.com/wcodesoft/please-rules/tree/ts)         | [`[TypeScript] v0.1.2`](https://github.com/wcodesoft/please-rules/releases/tag/ts-v0.1.2) | [TypeScript Rules Documentation](https://github.com/wcodesoft/please-rules/blob/ts/docs/ts/README.md)     | Hermetic Deno v2.2 toolchain, zero `node_modules`, ephemeral import maps, bundling, and testing.                    |
| **WIT (`///wit`)**       | [`wit`](https://github.com/wcodesoft/please-rules/tree/wit)       | [`[WIT] v0.1.0`](https://github.com/wcodesoft/please-rules/releases/tag/wit-v0.1.0)       | [WIT Rules](https://github.com/wcodesoft/please-rules/tree/wit)                                           | WebAssembly Interface Types (WIT) bundling and binding generators for Swift, Kotlin, Rust, Go, TS, C++, and Python. |

---

## Documentation

Detailed guides, language rule references, and architecture documentation:

- **[Documentation Portal](docs/README.md)**: Main documentation hub.
- **[Getting Started Guide](docs/getting-started.md)**: Project integration and
  setup.
- **[Multi-Language Gitflow Guide](docs/gitflow.md)**: Repository branching
  model, release tagging, and synchronization workflows.
- **Language Rulesets**:
  - **[Rust Documentation](https://github.com/wcodesoft/please-rules/blob/rust/docs/rust/README.md)**
    (`rust` branch)
  - **[Kotlin Documentation](https://github.com/wcodesoft/please-rules/blob/kotlin/docs/kotlin/README.md)**
    (`kotlin` branch)
  - **[Swift Documentation](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/README.md)**
    (`swift` branch)
  - **[TypeScript Documentation](https://github.com/wcodesoft/please-rules/blob/ts/docs/ts/README.md)**
    (`ts` branch)
  - **[WIT Documentation](https://github.com/wcodesoft/please-rules/tree/wit)**
    (`wit` branch)

---

## Building & Testing

To run all tests across the repository:

```bash
./pleasew test //...
```

To build all targets:

```bash
./pleasew build //...
```

---

## Development & IDE Setup (Go / VSCode)

`please-rules` builds its helper tools (such as `please_rust`, `please_kotlin`,
`please_swift`, `please_ts`) using Please's hermetic Go toolchain with
root-relative package import paths (e.g.
`import "tools/please_rust/testrunner"`).

For optimal VSCode editor integration without conflicting with external Go
toolchains, `.vscode/settings.json` is configured to disable background `gopls`
builds while preserving format on save.
