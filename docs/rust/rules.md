# Rule Reference

This document provides a comprehensive reference for all build rules provided by
the Please Rust plugin (`please-rules`).

To use these rules in your `BUILD` files, subinclude the plugin definitions:

```starlark
subinclude("///rust//build_defs:rust")
```

---

## `rust_library`

Compiles Rust source files into a static library artifact (`lib<crate>.rlib`).

### `rust_library` Example

```starlark
rust_library(
    name = "core_utils",
    srcs = [
        "lib.rs",
        "helpers.rs",
    ],
    crate_name = "core_utils",
    edition = "2024",
    deps = [
        "//third_party/rust:serde",
    ],
    visibility = ["PUBLIC"],
)
```

### `rust_library` Parameters

| Parameter    | Type   | Default                                | Description                                                                                                    |
| :----------- | :----- | :------------------------------------- | :------------------------------------------------------------------------------------------------------------- |
| `name`       | `str`  | _Required_                             | Name of the build target.                                                                                      |
| `srcs`       | `list` | `[]`                                   | Source files included in the compilation. If not explicitly set, defaults to `lib.rs` or first file in `srcs`. |
| `deps`       | `list` | `[]`                                   | List of dependency targets (`rust_library` or `rust_crate`).                                                   |
| `edition`    | `str`  | `CONFIG.RUST.DEFAULT_EDITION` (`2024`) | Rust edition (e.g. `"2021"`, `"2024"`).                                                                        |
| `flags`      | `list` | `[]`                                   | Additional flags passed directly to `rustc`.                                                                   |
| `crate_name` | `str`  | `name`                                 | Custom name for the compiled crate.                                                                            |
| `main`       | `str`  | `None`                                 | Explicit root entrypoint source file (e.g. `"src/lib.rs"`, `"lib.rs"`).                                        |
| `env`        | `dict` | `{}`                                   | Compile-time environment variables exported for `rustc` (`export KEY="val"`).                                  |
| `visibility` | `list` | `None`                                 | Target visibility list.                                                                                        |
| `test_only`  | `bool` | `False`                                | If `True`, target can only be depended on by test rules.                                                       |
| `labels`     | `list` | `["rust"]`                             | Labels associated with the target.                                                                             |

---

## `rust_bin`

Compiles Rust source files into an executable binary target.

### `rust_bin` Example

```starlark
rust_bin(
    name = "server",
    srcs = [
        "main.rs",
        "config.rs",
    ],
    edition = "2024",
    deps = [
        ":core_utils",
        "//third_party/rust:clap",
    ],
    visibility = ["PUBLIC"],
)
```

### `rust_bin` Parameters

| Parameter    | Type   | Default                                | Description                                                                          |
| :----------- | :----- | :------------------------------------- | :----------------------------------------------------------------------------------- |
| `name`       | `str`  | _Required_                             | Name of the executable target.                                                       |
| `srcs`       | `list` | `[]`                                   | Source files included in compilation. Defaults to `main.rs` or first file in `srcs`. |
| `deps`       | `list` | `[]`                                   | List of target dependencies.                                                         |
| `edition`    | `str`  | `CONFIG.RUST.DEFAULT_EDITION` (`2024`) | Rust edition.                                                                        |
| `flags`      | `list` | `[]`                                   | Additional compiler flags for `rustc`.                                               |
| `crate_name` | `str`  | `name`                                 | Crate name for the binary target.                                                    |
| `main`       | `str`  | `None`                                 | Explicit root entrypoint source file (e.g. `"src/main.rs"`, `"main.rs"`).            |
| `env`        | `dict` | `{}`                                   | Compile-time environment variables exported for `rustc` (`export KEY="val"`).        |
| `visibility` | `list` | `None`                                 | Target visibility list.                                                              |
| `test_only`  | `bool` | `False`                                | If `True`, target can only be depended on by test rules.                             |
| `labels`     | `list` | `["rust"]`                             | Labels associated with the target.                                                   |

---

## `rust_test`

Compiles Rust source files with test configuration (`--test`) and runs the
unit/integration tests using Please's test runner, outputting standard JUnit XML
results.

### `rust_test` Example

```starlark
rust_test(
    name = "core_utils_test",
    srcs = [
        "lib.rs",
        "helpers.rs",
    ],
    crate_name = "core_utils",
    edition = "2024",
    deps = [
        ":core_utils",
    ],
)
```

### `rust_test` Parameters

| Parameter    | Type   | Default                                | Description                                                             |
| :----------- | :----- | :------------------------------------- | :---------------------------------------------------------------------- |
| `name`       | `str`  | _Required_                             | Name of the test target.                                                |
| `srcs`       | `list` | `[]`                                   | Source files included in test compilation.                              |
| `deps`       | `list` | `[]`                                   | Target dependencies required for the test.                              |
| `edition`    | `str`  | `CONFIG.RUST.DEFAULT_EDITION` (`2024`) | Rust edition.                                                           |
| `flags`      | `list` | `[]`                                   | Extra flags passed to `rustc`.                                          |
| `crate_name` | `str`  | `name`                                 | Crate name under test.                                                  |
| `main`       | `str`  | `None`                                 | Explicit root entrypoint source file (e.g. `"src/lib.rs"`, `"lib.rs"`). |
| `env`        | `dict` | `{}`                                   | Environment variables exported for test compilation and test execution. |
| `visibility` | `list` | `None`                                 | Target visibility list.                                                 |
| `labels`     | `list` | `["rust"]`                             | Labels for the target.                                                  |
| `data`       | `list` | `None`                                 | Runtime data files needed by test execution.                            |
| `size`       | `str`  | `"medium"`                             | Please test size category (`"small"`, `"medium"`, `"large"`).           |
| `timeout`    | `int`  | `0`                                    | Execution timeout in seconds (`0` indicates default timeout).           |
| `flaky`      | `bool` | `False`                                | Mark target as flaky for automatic retries.                             |

---

## `rust_crate`

Hermetically downloads third-party `.crate` archives directly from `crates.io`,
verifies their SHA-256 digest, extracts crate metadata, optionally compiles
embedded C sources, and compiles directly with `rustc` into `.rlib` or `.so`
artifacts—completely without Cargo at build time.

### `rust_crate` Example

```starlark
# Standard library crate
rust_crate(
    name = "itoa",
    version = "1.0.14",
    sha256 = "d75a2a4b1b190afb6f5425f10f6a8f959d2ea0b9c2b1d79553551850539e4674",
)

# Crate with features and dependencies
rust_crate(
    name = "serde",
    version = "1.0.217",
    sha256 = "02fc4265df13d6fa1d00ecff087228cc0a2b5f3c0e87e258d8b94a156e984c70",
    features = ["derive", "std"],
    deps = [":serde_derive"],
)

# Procedural macro crate (outputs .so)
rust_crate(
    name = "serde_derive",
    version = "1.0.217",
    sha256 = "5a9bf7cf98d04a2b28aead066b7496853d4779c9cc183c440dbac457641e19a0",
    proc_macro = True,
    deps = [":syn", ":quote", ":proc-macro2"],
)

# Crate with native C sources (e.g. Tree-sitter parsers)
rust_crate(
    name = "tree-sitter-c",
    crate_name = "tree_sitter_c",
    version = "0.23.4",
    sha256 = "afd2b1bf1585dc2ef6d69e87d01db8adb059006649dd5f96f31aa789ee6e9c71",
    c_srcs = ["src/parser.c"],
    c_hdrs = ["src"],
    deps = [":tree-sitter-language"],
)
```

### Computing Checksums

To compute the SHA-256 digest for any crate version from `crates.io`:

```bash
plz-out/bin/tools/please_rust/please_rust hash --crate <name> --version <version>
```

### `rust_crate` Parameters

| Parameter    | Type   | Default      | Description                                                                                               |
| :----------- | :----- | :----------- | :-------------------------------------------------------------------------------------------------------- |
| `name`       | `str`  | _Required_   | Target name in the `BUILD` file.                                                                          |
| `version`    | `str`  | _Required_   | Exact package version on `crates.io`.                                                                     |
| `sha256`     | `str`  | _Required_   | SHA-256 hex digest of the `.crate` tarball from `crates.io`.                                              |
| `crate_name` | `str`  | `""`         | Crate name override (defaults to `name`). Hyphens (`-`) are automatically converted to underscores (`_`). |
| `features`   | `list` | `[]`         | List of enabled Cargo features (each emitted as `--cfg feature="<name>"` to `rustc`).                     |
| `deps`       | `list` | `[]`         | Dependency targets required by this crate.                                                                |
| `c_srcs`     | `list` | `[]`         | C source files compiled into a static library (`.a`) using the system C compiler (`cc`).                  |
| `c_hdrs`     | `list` | `[]`         | Header include directories (`-I`) for C source compilation.                                               |
| `flags`      | `list` | `[]`         | Additional flags passed directly to `rustc` (e.g. `--cfg=...`).                                           |
| `proc_macro` | `bool` | `False`      | Set to `True` if the crate is a procedural macro (outputs `.so` shared library).                          |
| `meta`       | `bool` | `False`      | If `True`, creates an aggregator/virtual target grouping rule (`filegroup`).                              |
| `license`    | `str`  | `""`         | Informational SPDX license string metadata.                                                               |
| `repository` | `str`  | `""`         | Informational source repository URL metadata.                                                             |
| `visibility` | `list` | `["PUBLIC"]` | Target visibility list.                                                                                   |
