# Please Rules (`please-rules`)

A curated collection of modern, hermetic build rules and plugins for the
[Please](https://please.build) build system.

This repository provides language toolchains and build definitions designed for
hermeticity, reproducible builds, and minimal host dependencies.

---

## Getting Started

To integrate and use rules from this repository in your Please project, see the
**[Getting Started & Integration Guide](docs/getting-started/README.md)**.

---

## Documentation

Detailed guides, language rule references, and architecture documentation are
organized in the [`docs/`](docs/) directory:

- **[Documentation Portal](docs/README.md)**: Main documentation hub.
- **[Getting Started Guide](docs/getting-started/README.md)**: Project
  integration and setup.
- **[Rust Rules Documentation](docs/rust/README.md)**: Rust architecture, rules,
  and usage.

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

`please-rules` builds its helper tools (such as `please_rust`) using Please's
hermetic Go toolchain with root-relative package import paths (e.g.
`import "tools/please_rust/testrunner"`).

For optimal VSCode editor integration without conflicting with external Go
toolchains, `.vscode/settings.json` is configured to disable background `gopls`
builds while preserving format on save.
