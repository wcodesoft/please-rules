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

## Hermetic Toolchain Setup (Recommended)

To achieve reproducible builds across all developer workstations and CI agents
without requiring `rustup`, host `rustc`, or passing environment variables:

### 1. Declare the Toolchain Target

In a shared build file (e.g. `third_party/rust/BUILD` or
`build_defs/rust/BUILD`):

```starlark
subinclude("///rust//build_defs:rust")

rust_toolchain(
    name = "toolchain",
    version = "1.85.0",
)
```

### 2. Bind the Toolchain in `.plzconfig`

Set `RustcTool` in `.plzconfig` to the toolchain's `rustc` entry point:

```ini
[Plugin "rust"]
Target = //plugins:rust
RustcTool = //third_party/rust:toolchain|rustc
```

Please will download the standalone compiler and standard library archives from
`static.rust-lang.org`, verify the SHA-256 checksums, unpack the sysroot, and
supply `$TOOLS_RUSTC` to all compilation actions automatically.

---

## Configuration Options

The Rust plugin supports several configuration keys in `.plzconfig`:

```ini
[Plugin "rust"]
Target = //plugins:rust

# (Recommended) Target entry point to hermetic toolchain
RustcTool = //third_party/rust:toolchain|rustc

# (Alternative) Host binary path override
; RustcTool = /usr/bin/rustc

# (Optional) Custom path to please_rust helper tool target
PleaseRustTool = //tools/please_rust

# Default Rust edition for rules when not specified in BUILD file
DefaultEdition = 2021

# Coverage instrumentation enabled by default
Coverage = true

# LLVM profdata and cov tools (for coverage reporting)
LlvmProfdataTool = //third_party/rust:toolchain|llvm-profdata
LlvmCovTool = //third_party/rust:toolchain|llvm-cov
```

### Options Summary

| Config Key         | Option Name          | Default               | Description                                                                                     |
| :----------------- | :------------------- | :-------------------- | :---------------------------------------------------------------------------------------------- |
| `PleaseRustTool`   | `please_rust_tool`   | `//tools/please_rust` | Path or target label for the `please_rust` Go helper tool.                                      |
| `RustcTool`        | `rustc_tool`         | `rustc`               | Target entry point (e.g. `//pkg:toolchain\|rustc`) or executable command/path for the compiler. |
| `DefaultEdition`   | `default_edition`    | `2021`                | Default Rust edition (e.g. `2021`, `2024`).                                                     |
| `Coverage`         | `coverage`           | `true`                | Whether to compile Rust targets with coverage instrumentation.                                  |
| `LlvmProfdataTool` | `llvm_profdata_tool` | `""`                  | Path or target label for `llvm-profdata`.                                                       |
| `LlvmCovTool`      | `llvm_cov_tool`      | `""`                  | Path or target label for `llvm-cov`.                                                            |

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

---

## Code Coverage (`plz cover`)

The Rust rules provide full code coverage instrumentation and reporting out of
the box using LLVM source-based code coverage (`-C instrument-coverage`).

### 1. Configure Coverage in `.plzconfig`

Ensure file extension `.rs` is tracked under `[cover]` and optionally configure
coverage options under `[Plugin "rust"]`:

```ini
[cover]
fileextension = .rs

[Plugin "rust"]
Target = //plugins:rust
# (Optional) Enable/disable coverage instrumentation globally (default: true)
Coverage = true

# (Optional) Explicit overrides for llvm-profdata and llvm-cov
; LlvmProfdataTool = //third_party/rust:toolchain|llvm-profdata
; LlvmCovTool = //third_party/rust:toolchain|llvm-cov
```

### 2. Toolchain Coverage Discovery

- **Hermetic Toolchain (`rust_toolchain`)**: When using `rust_toolchain`, the
  corresponding `llvm-tools` component archive is automatically downloaded and
  unpacked alongside `rustc` and `rust-std`. The `llvm-profdata` and `llvm-cov`
  entry points are exported as `:toolchain|llvm-profdata` and
  `:toolchain|llvm-cov`.
- **Host Toolchain**: If no explicit toolchain is configured, `please_rust`
  automatically discovers `llvm-profdata` and `llvm-cov` from your system:
  1. Standard system `PATH` (e.g. `llvm-profdata`, `llvm-cov`)
  2. Versioned LLVM directories (e.g. `/usr/lib/llvm-*/bin`)
  3. Active `rustup` toolchain
     (`$(rustc --print sysroot)/lib/rustlib/<target>/bin`)

### 3. Running Coverage

Run coverage across all tests:

```bash
./pleasew cover //...
```

Run coverage on a specific test target:

```bash
./pleasew cover //test/lib:lib_test
```

Display detailed, line-by-line covered source listings:

```bash
./pleasew cover -l //test/lib:lib_test
```

### 4. Report Output

- **Location**: Please merges the per-test results into
  `plz-out/log/coverage.xml` (Cobertura XML) and `plz-out/log/coverage.json`
  (per-file line markers plus per-file and per-directory percentages).
- **Contents**: line coverage only. Function and branch data are in the raw lcov
  export described below.
- **Missing `[cover]` setting**: without `fileextension = .rs` the report is
  silently `Total coverage: No data`.

### 5. Raw lcov Export (functions and branches)

Each `rust_test` also keeps the `llvm-cov` report as lcov, with the function
(`FN`/`FNDA`) and branch (`BRDA`) records that `coverage.xml` and
`coverage.json` do not carry:

- **Location**: `plz-out/bin/<package>/<target name>.lcov`, for example
  `plz-out/bin/test/rust/branches/branches_test.lcov`. It is declared as a test
  output, so it is written next to the test binary.
- **Content**: lcov with repository-relative paths; toolchain and registry
  sources are excluded. The file is empty unless the test ran under `plz cover`.
- **Functions**: `FN`/`FNDA` records. Names are rustc v0-mangled (`_R...`);
  demangle them with `rustfilt` for display.
- **Partial lines**: stable `rustc` emits no branch regions, so `llvm-cov` has
  no branch records of its own. Instead, a `BRDA` record is synthesized for
  every line that ran but contains a sub-line region that never did, such as the
  right-hand side of `a && b` or the body of a one-line `if`: one arm per region
  starting on the line, in column order, with its count. This is a heuristic,
  not source-level branches. If the arms of a branch are on separate lines (a
  normal multi-line `if`/`else`), the untaken side shows up as an uncovered
  line, not as a partial one. Lines whose sub-line regions all ran carry no
  branch record.
- **Lifetime**: the file reflects the last test run that actually executed. Read
  it right after `./pleasew cover --rerun <target>`; do not assume it is
  recreated when Please reuses a cached result.

### 6. Per-Rule Coverage Control

You can enable or disable coverage instrumentation on individual rules:

```starlark
rust_library(
    name = "my_lib",
    srcs = ["lib.rs"],
    coverage = True, # Explicitly enable instrumentation
)

rust_test(
    name = "my_test",
    srcs = ["test.rs"],
    deps = [":my_lib"],
    coverage = True,
)
```
