# Please Rules (`please-rules`)

A curated collection of modern, hermetic build rules and plugins for the
[Please](https://please.build) build system.

This repository provides language toolchains and build definitions designed for
hermeticity, reproducible builds, and minimal host dependencies.

---

## Available Rulesets

### [Rust Rules](docs/rust/README.md) (`///rust`)

First-class, hermetic support for building, testing, and managing Rust crates:

- **`rust_library`**: Compiles Rust source files into `.rlib` libraries.
- **`rust_bin`**: Compiles native Rust executables.
- **`rust_test`**: Runs unit and integration tests with automatic test parsing
  and JUnit XML reporting.
- **`rust_crate`**: Hermetically downloads and builds third-party crates
  directly from `crates.io` with mandatory SHA-256 integrity verification,
  compiling directly with `rustc` without Cargo at build time.
- **Native C Extension Support**: Seamlessly compiles C sources (`c_srcs`) for
  grammar crates (e.g., Tree-sitter) into static archives without `build.rs`.
- **Dedicated CLI Backend**: Uses `please_rust` for compilation orchestration,
  checksumming, C compilation, and JUnit test result translation.

👉 **[Rust Documentation & Rule Reference](docs/rust/README.md)**

---

## Getting Started

To consume rules from this repository in your Please project, declare the plugin
dependency in `plugins/BUILD`:

```starlark
plugin_repo(
    name = "rust",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "v0.3.0",
)
```

Configure the plugin in your `.plzconfig`:

```ini
[Plugin "rust"]
Target = //plugins:rust

[build]
passenv = PATH, HOME
```

Subinclude the rules in your `BUILD` files:

```starlark
subinclude("///rust//build_defs:rust")
```

---

## Repository Structure

```
please-rules/
├── build_defs/          # Starlark rule definitions by language (e.g. rust/)
├── tools/               # Native compiler orchestrators and helper CLI binaries
├── plugins/             # Plugin repository declarations
├── docs/                # Documentation hub and language-specific guides
│   └── rust/            # Rust architecture, rule reference, and guides
└── test/                # End-to-end integration test suites
```

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

## Documentation

Detailed design documents, rule references, usage guides, and architectural
details are organized in the [`docs/`](docs/) directory:

- **[Documentation Portal](docs/README.md)**
- **[Rust Architecture & Internal Design](docs/rust/architecture.md)**
- **[Rust Rule Reference](docs/rust/rules.md)**
- **[Rust Dependencies & Toolchains](docs/rust/dependencies.md)**
- **[Rust Usage Guide](docs/rust/usage.md)**

---

## Code Quality (Veritas)

This repository integrates with [Veritas](https://github.com/walterjgsp/veritas)
for static code analysis and automated test telemetry.

### Running Analysis Locally

```bash
# Analyze repository and show summary
veritas analyze --summary-only .

# Check against quality threshold violations
veritas analyze -V .
```

### Telemetry & Reporting

- `./pleasew test` automatically uploads test results to the Veritas dashboard.
  To disable, run with `VERITAS_NO_HOOK=1`.
- A Git post-commit hook in `.git/hooks/post-commit` automatically records code
  quality snapshots on every commit.

---

## Development & IDE Setup (Go / VSCode)

`please-rules` builds its helper tools (such as `please_rust`) using Please's
hermetic Go toolchain with root-relative package import paths (e.g.
`import "tools/please_rust/testrunner"`).

For optimal VSCode editor integration without conflicting with external Go
toolchains, `.vscode/settings.json` is configured to disable background `gopls`
builds while preserving format on save.
