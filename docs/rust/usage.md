# Usage & Configuration Guide

This guide covers how to set up, configure, and use the Please Rust plugin
(`please-rules`) in your project repository.

---

## Plugin Setup

### 1. Register the Plugin in `.plzconfig`

Add the Rust plugin configuration to your repository's `.plzconfig` file:

```ini
[Plugin "rust"]
Target = //plugins:rust
```

### 2. Subinclude Build Definitions

To use Rust build rules inside a `BUILD` file, subinclude the rule definitions:

```starlark
subinclude("///rust//build_defs:rust")
```

---

## Configuration Options

The Rust plugin supports several configuration keys in `.plzconfig`:

```ini
[Plugin "rust"]
Target = //plugins:rust

# (Optional) Custom path to please_rust helper tool target
PleaseRustTool = //tools/please_rust

# (Optional) Custom path to rustc binary
RustcTool = rustc

# (Optional) Custom path to cargo binary
CargoTool = cargo

# Default Rust edition for rules when not specified in BUILD file
DefaultEdition = 2024
```

### Options Summary

| Config Key       | Option Name        | Default               | Description                                                |
| :--------------- | :----------------- | :-------------------- | :--------------------------------------------------------- |
| `PleaseRustTool` | `please_rust_tool` | `//tools/please_rust` | Path or target label for the `please_rust` Go helper tool. |
| `RustcTool`      | `rustc_tool`       | `rustc`               | Path or command name for `rustc`.                          |
| `CargoTool`      | `cargo_tool`       | `cargo`               | Path or command name for `cargo` (legacy fallback).        |
| `DefaultEdition` | `default_edition`  | `2024`                | Default Rust edition (e.g. `2021`, `2024`).                |

---

## Project Structure Examples

Source files are placed directly in the same directory as their corresponding
`BUILD` file, without requiring a `src/` subfolder.

### Single Crate Repository Layout

```txt
my_project/
├── .plzconfig
├── BUILD
├── lib.rs
└── main.rs
```

`BUILD`:

```starlark
subinclude("///rust//build_defs:rust")

rust_library(
    name = "my_lib",
    srcs = ["lib.rs"],
)

rust_bin(
    name = "app",
    srcs = ["main.rs"],
    deps = [":my_lib"],
)

rust_test(
    name = "my_lib_test",
    srcs = ["lib.rs"],
    deps = [":my_lib"],
)
```

### Multi-Crate Workspace Layout

```txt
my_workspace/
├── .plzconfig
├── BUILD
├── third_party/
│   └── rust/
│       └── BUILD
├── crates/
│   ├── common/
│   │   ├── BUILD
│   │   └── lib.rs
│   └── service/
│       ├── BUILD
│       └── main.rs
```

`third_party/rust/BUILD`:

```starlark
subinclude("///rust//build_defs:rust")

rust_crate(
    name = "serde",
    version = "1.0.217",
    sha256 = "02fc4265df13d6fa1d00ecff087228cc0a2b5f3c0e87e258d8b94a156e984c70",
    features = ["derive", "std"],
)

rust_crate(
    name = "clap",
    version = "4.5.31",
    sha256 = "027bb0d98429ae334a8698531da7077bdf906419543a35a55c2cb1b66437d767",
    features = ["derive"],
)
```

`crates/common/BUILD`:

```starlark
subinclude("///rust//build_defs:rust")

rust_library(
    name = "common",
    srcs = ["lib.rs"],
    deps = [
        "//third_party/rust:serde",
    ],
    visibility = ["//crates/..."],
)
```

`crates/service/BUILD`:

```starlark
subinclude("///rust//build_defs:rust")

rust_bin(
    name = "service",
    srcs = ["main.rs"],
    deps = [
        "//crates/common",
        "//third_party/rust:clap",
    ],
)
```

---

## Common Commands

### Building Targets

Build all targets in the repository:

```bash
./pleasew build //...
```

Build a specific binary target:

```bash
./pleasew build //crates/service
```

### Running Executables

Run a built binary:

```bash
./pleasew run //crates/service
```

### Running Tests

Run all unit tests across the repository:

```bash
./pleasew test //...
```

Run a specific test target:

```bash
./pleasew test //crates/common:common_test
```
