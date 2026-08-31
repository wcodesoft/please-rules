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

## Development & IDE Setup (Go / VSCode)

Please.build calculates package import paths relative to the repository root
when `ImportPath` is not set or set to `""`.

In this repository:

- `tools/please_rust/...` imports packages using root-relative paths:
  ```go
  import "tools/please_rust/testrunner"
  ```
- VSCode's Go language server (`gopls`) looks for a Go module matching the root
  import path. A minimal `go.mod` in the repository root defines:
  ```
  module tools
  go 1.22
  ```
  This allows `tools/...` imports to resolve directly without requiring a full
  URL or domain prefix in both Please and VSCode.

If VSCode displays `cannot find package ... in GOROOT`:

1. Ensure `go.mod` is present in the repository root with `module tools`.
2. Reload the VSCode window or run `Ctrl+Shift+P` ->
   `Go: Restart Language Server`.

---

## Documentation

Detailed design documents, implementation plans, and architectural decisions are
located in the [`docs/`](docs/) directory:

- [Rust Rules Plugin Plan](docs/rust_rules_plugin_plan.md)
