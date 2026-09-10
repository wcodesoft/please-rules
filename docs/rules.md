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
    edition = "2021",
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
| `edition`    | `str`  | `CONFIG.RUST.DEFAULT_EDITION` (`2021`) | Rust edition (e.g. `"2021"`, `"2024"`).                                                                        |
| `flags`      | `list` | `[]`                                   | Additional flags passed directly to `rustc`.                                                                   |
| `crate_name` | `str`  | `name`                                 | Custom name for the compiled crate.                                                                            |
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
    edition = "2021",
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
| `edition`    | `str`  | `CONFIG.RUST.DEFAULT_EDITION` (`2021`) | Rust edition.                                                                        |
| `flags`      | `list` | `[]`                                   | Additional compiler flags for `rustc`.                                               |
| `crate_name` | `str`  | `name`                                 | Crate name for the binary target.                                                    |
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
    edition = "2021",
    deps = [
        ":core_utils",
    ],
)
```

### `rust_test` Parameters

| Parameter    | Type   | Default                                | Description                                                   |
| :----------- | :----- | :------------------------------------- | :------------------------------------------------------------ |
| `name`       | `str`  | _Required_                             | Name of the test target.                                      |
| `srcs`       | `list` | `[]`                                   | Source files included in test compilation.                    |
| `deps`       | `list` | `[]`                                   | Target dependencies required for the test.                    |
| `edition`    | `str`  | `CONFIG.RUST.DEFAULT_EDITION` (`2021`) | Rust edition.                                                 |
| `flags`      | `list` | `[]`                                   | Extra flags passed to `rustc`.                                |
| `crate_name` | `str`  | `name`                                 | Crate name under test.                                        |
| `visibility` | `list` | `None`                                 | Target visibility list.                                       |
| `labels`     | `list` | `["rust"]`                             | Labels for the target.                                        |
| `data`       | `list` | `None`                                 | Runtime data files needed by test execution.                  |
| `size`       | `str`  | `"medium"`                             | Please test size category (`"small"`, `"medium"`, `"large"`). |
| `timeout`    | `int`  | `0`                                    | Execution timeout in seconds (`0` indicates default timeout). |
| `flaky`      | `bool` | `False`                                | Mark target as flaky for automatic retries.                   |

---

## `rust_crate`

Fetches and builds third-party crates from `crates.io` using Cargo in an
isolated temporary build environment, producing `.rlib` or `.so` artifacts.

### `rust_crate` Example

```starlark
rust_crate(
    name = "serde",
    version = "1.0.197",
    features = ["derive", "std"],
)

rust_crate(
    name = "serde_derive",
    version = "1.0.197",
    proc_macro = True,
)
```

### `rust_crate` Parameters

| Parameter    | Type   | Default      | Description                                                                                               |
| :----------- | :----- | :----------- | :-------------------------------------------------------------------------------------------------------- |
| `name`       | `str`  | _Required_   | Rule name in the `BUILD` file.                                                                            |
| `crate_name` | `str`  | `""`         | Crate name override (defaults to `name`). hyphens (`-`) are automatically converted to underscores (`_`). |
| `pkg_name`   | `str`  | `""`         | Cargo package name on crates.io (defaults to `name`).                                                     |
| `version`    | `str`  | `""`         | Exact package version on crates.io.                                                                       |
| `features`   | `list` | `[]`         | List of enabled Cargo features.                                                                           |
| `deps`       | `list` | `[]`         | Dependency targets required by this crate.                                                                |
| `proc_macro` | `bool` | `False`      | Set to `True` if the crate is a procedural macro (outputs `.so` shared library).                          |
| `meta`       | `bool` | `False`      | If `True` or if `version` is empty, creates a target grouping rule (`filegroup`).                         |
| `license`    | `str`  | `""`         | License string metadata.                                                                                  |
| `repository` | `str`  | `""`         | Source repository URL metadata.                                                                           |
| `visibility` | `list` | `["PUBLIC"]` | Target visibility list.                                                                                   |
