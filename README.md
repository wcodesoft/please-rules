# Rust Rules for Please (`please-rules`)

A [Please](https://please.build) build system plugin providing first-class build
and test rules for Rust.

## Features

- **`rust_toolchain`**: Downloads, verifies, and packages official Rust compiler distributions hermetically, eliminating external host toolchain dependencies (`rustup`, system `rustc`) and avoiding `passenv = PATH, HOME`.
- **`rust_library`**: Compiles Rust source files into `.rlib` libraries.
- **`rust_bin`**: Builds native Rust binary executables.
- **`rust_test`**: Runs unit and integration tests with automatic test parsing
  and JUnit XML results generation.
- **`rust_crate`**: Hermetically downloads and compiles third-party crates
  directly from `crates.io` with mandatory SHA-256 integrity verification,
  compiling directly with `rustc` without Cargo at build time.
- **Native C Extension Support**: Seamlessly compiles C sources (`c_srcs`) into
  static archives for grammar crates like Tree-sitter without `build.rs`.
- **Hermetic Go Backend**: Implements a dedicated multi-subcommand CLI
  (`please_rust`) for compilation orchestration, crate downloading, SHA-256
  checksumming, C compilation, and JUnit test results translation.
- **Flexible Layout**: Works seamlessly with standard `src/lib.rs` /
  `src/main.rs` layouts or flat file structures (`lib.rs`, `main.rs`,
  `<name>.rs`). Default edition is `2024`.

---

## Installation & Setup

Add the plugin configuration to your repo's `.plzconfig`:

```ini
[Plugin "rust"]
Target = //plugins:rust
RustcTool = //build_defs/rust:toolchain|rustc
```

In your `BUILD` files:

```starlark
subinclude("///rust//build_defs:rust")

rust_toolchain(
    name = "toolchain",
    version = "1.85.0",
    visibility = ["PUBLIC"],
)
```

---

## Migration Guide: Host to Hermetic Toolchain

If your repository currently relies on a host-installed Rust compiler via `rustup` or `PATH`:

1. **Instantiate `rust_toolchain`**:
   In your root or toolchain `BUILD` file (e.g. `build_defs/rust/BUILD`):
   ```starlark
   subinclude("///rust//build_defs:rust")

   rust_toolchain(
       name = "toolchain",
       version = "1.85.0",
       visibility = ["PUBLIC"],
   )
   ```

2. **Configure `.plzconfig`**:
   Bind the plugin's `RustcTool` setting to the hermetic toolchain label:
   ```ini
   [Plugin "rust"]
   Target = //plugins:rust
   RustcTool = //build_defs/rust:toolchain|rustc
   ```

3. **Remove Host Environment Leaks**:
   Remove `passenv = PATH, HOME` from your `.plzconfig` `[build]` section. Builds will now execute in complete sandbox isolation using the downloaded hermetic sysroot inside `plz-out/`.

---

## Rule Reference

### `rust_toolchain`

Fetches official standalone Rust toolchain distributions from `static.rust-lang.org`, verifies integrity digests, assembles a hermetic sysroot in `plz-out/`, and exposes `rustc` as a Please target.

```starlark
rust_toolchain(
    name = "toolchain",
    version = "1.85.0",
    visibility = ["PUBLIC"],
)
```

### `rust_library`

Compiles a Rust library crate.

```starlark
rust_library(
    name = "my_lib",
    srcs = ["lib.rs"],
    crate_name = "my_lib",
    edition = "2024",
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
    edition = "2024",
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
    edition = "2024",
    deps = [
        ":my_lib",
    ],
)
```

### `rust_crate`

Hermetically downloads and builds third-party crates without Cargo. `version` and
`sha256` are mandatory for supply-chain security:

```starlark
rust_crate(
    name = "itoa",
    version = "1.0.14",
    sha256 = "d75a2a4b1b190afb6f5425f10f6a8f959d2ea0b9c2b1d79553551850539e4674",
)

# Crate with optional features and dependencies
rust_crate(
    name = "serde",
    version = "1.0.217",
    sha256 = "02fc4265df13d6fa1d00ecff087228cc0a2b5f3c0e87e258d8b94a156e984c70",
    features = ["derive", "std"],
    deps = [":serde_derive"],
)

# Procedural macro crate
rust_crate(
    name = "serde_derive",
    version = "1.0.217",
    sha256 = "5a9bf7cf98d04a2b28aead066b7496853d4779c9cc183c440dbac457641e19a0",
    proc_macro = True,
    deps = [":syn", ":quote", ":proc-macro2"],
)

# Crate with C native sources (e.g. tree-sitter grammars)
rust_crate(
    name = "tree-sitter-c",
    version = "0.23.4",
    sha256 = "afd2b1bf1585dc2ef6d69e87d01db8adb059006649dd5f96f31aa789ee6e9c71",
    c_srcs = ["src/parser.c"],
    c_hdrs = ["src"],
    deps = [":tree-sitter-language"],
)
```

To compute the SHA-256 checksum for any crate on `crates.io`:

```bash
plz-out/bin/tools/please_rust/please_rust hash --crate itoa --version 1.0.14
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
