# Rust Rules for Please (`please-rules`)

A [Please](https://please.build) build system plugin providing first-class build
and test rules for Rust.

## Features

- **`rust_library`**: Compiles Rust source files into `.rlib` libraries.
- **`rust_bin`**: Builds native Rust binary executables.
- **`rust_test`**: Runs unit and integration tests with automatic test parsing
  and JUnit XML results generation.
- **`rust_crate`**: Fetches and compiles third-party crates from `crates.io`
  using Cargo.
- **Hermetic Go Backend**: Implements a dedicated tool (`please_rust`) built via
  the Please Go plugin for dependency resolution, compilation orchestration,
  test parsing, and crate fetching.
- **Flexible Layout**: Works seamlessly with standard `src/lib.rs` /
  `src/main.rs` layouts or flat file structures (`lib.rs`, `main.rs`,
  `<name>.rs`).

---

## Installation & Setup

Add the plugin configuration to your repo's `.plzconfig`:

```ini
[Plugin "rust"]
Target = //plugins:rust
```

In your `BUILD` files:

```starlark
subinclude("///rust//build_defs:rust")
```

---

## Rule Reference

### `rust_library`

Compiles a Rust library crate.

```starlark
rust_library(
    name = "my_lib",
    srcs = ["lib.rs"],
    crate_name = "my_lib",
    edition = "2021",
    deps = [
        "//third_party/rust:serde",
    ],
    visibility = ["PUBLIC"],
)
```

### `rust_bin`

Compiles an executable binary.

```starlark
rust_bin(
    name = "my_binary",
    srcs = ["main.rs"],
    edition = "2021",
    deps = [
        ":my_lib",
    ],
)
```

### `rust_test`

Compiles and executes tests, reporting individual test cases to Please.

```starlark
rust_test(
    name = "my_test",
    srcs = ["lib.rs"],
    crate_name = "my_lib",
    edition = "2021",
    deps = [
        ":my_lib",
    ],
)
```

### `rust_crate`

Fetches and builds third-party crates:

```starlark
rust_crate(
    name = "itoa",
    version = "1.0.10",
    features = ["std"],
)
```

---

## Building & Testing

To run all tests in the repository:

```bash
./pleasew test //...
```

To build all targets:

```bash
./pleasew build //...
```

---

## Code Quality (Veritas)

This repository uses [Veritas](https://github.com/walterjgsp/veritas) for static
code analysis (Cognitive Complexity, Cyclomatic Complexity, LCOM4, and Code
Duplication) and automated test reporting.

### Configuration

Veritas is configured via [`.veritas/config.json`](.veritas/config.json):

```json
{
  "project": "please-rules",
  "server": "http://localhost:8080",
  "test_results_path": "plz-out/log/test_results.xml",
  "thresholds": {
    "cognitive_complexity": 15,
    "cyclomatic_complexity": 10,
    "lcom4": 1,
    "duplication_pct": 5.0
  }
}
```

To initialize or reconfigure Veritas with post-commit hooks:

```bash
veritas init --project please-rules --server http://localhost:8080 --install-hooks
```

### Static Analysis

Run code quality analysis locally across all files:

```bash
# Analyze repository and show summary
veritas analyze --summary-only .

# Check for quality threshold violations
veritas analyze -V .
```

### Automatic Test Reporting & Snapshots

- `./pleasew test` automatically uploads test results to the Veritas dashboard
  (`http://localhost:8080`). To disable, set `VERITAS_NO_HOOK=1`.
- A Git post-commit hook in `.git/hooks/post-commit` automatically uploads code
  quality snapshots on every commit in the background. You can also run
  `veritas upload-analysis` manually.

---

## Development & IDE Setup (Go / VSCode)

Please.build manages the Go toolchain and builds Go targets hermetically using
root-relative package import paths (e.g.,
`import "tools/please_rust/testrunner"`).

Because Please manages the builds rather than the native Go CLI/`go.mod`,
`.vscode/settings.json` is configured to disable background `gopls` / `go build`
checking while preserving formatting on save:

```json
{
  "editor.rulers": [80],
  "editor.formatOnSave": true,
  "[markdown]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  },
  "[go]": {
    "editor.formatOnSave": true,
    "editor.defaultFormatter": "golang.go"
  },
  "go.useLanguageServer": false,
  "go.buildOnSave": "off",
  "go.vetOnSave": "off",
  "go.lintOnSave": "off",
  "files.exclude": {
    "**/plz-out": true,
    "**/.please": true
  },
  "search.exclude": {
    "**/plz-out": true,
    "**/.please": true
  }
}
```

---

## Documentation

Detailed design documents, rule references, usage guides, and architectural
details are located in the [`docs/`](docs/) directory:

- [Documentation Index](docs/README.md)
- [Architecture & Design](docs/architecture.md)
- [Rule Reference](docs/rules.md)
- [Toolchain Dependencies](docs/dependencies.md)
- [Usage Guide](docs/usage.md)
