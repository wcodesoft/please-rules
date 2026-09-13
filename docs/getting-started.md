# Getting Started with `please-rules`

This guide explains how to integrate and consume build rules from `please-rules`
in your [Please](https://please.build) project.

---

## 1. Declaring the Plugin Dependency

In your project's `plugins/BUILD` file, declare the plugin repository using
`plugin_repo`:

```starlark
plugin_repo(
    name = "rust",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "rust-v0.4.0",     # or revision = "rust"
)
```

`please-rules` uses a **[Hub-and-Spoke Gitflow](gitflow.md)** where each
language ruleset (Rust, Kotlin, etc.) lives on its own dedicated branch.
Consumers specify the language branch or language release tag (e.g.
`rust-v0.4.0`) in `revision`.

---

## 2. Configuring `.plzconfig`

### Option A: Hermetic Toolchain (Recommended)

When using `rust_toolchain`, Please downloads and manages the compiler
hermetically. No host tools or environment variables need to be leaked:

```ini
; Please configuration file

[Plugin "rust"]
Target = //plugins:rust
RustcTool = //third_party/rust:toolchain|rustc
```

Where `//third_party/rust:toolchain` is defined in `third_party/rust/BUILD`:

```starlark
subinclude("///rust//build_defs:rust")

rust_toolchain(
    name = "toolchain",
    version = "1.85.0",
)
```

### Option B: Host System Toolchain

If using host-installed compilers or dispatchers (such as `rustup`):

```ini
; Please configuration file

[Plugin "rust"]
Target = //plugins:rust

[build]
passenv = PATH, HOME
```

Passing `PATH` and `HOME` ensures that Please's sandboxed build actions can
resolve host `rustc` and Cargo home directories without hardcoding paths.

---

## 3. Subincluding Rules in `BUILD` Files

In any package where you want to use the rules, subinclude the rule definitions:

```starlark
subinclude("///rust//build_defs:rust")
```

Alternatively, you can preload the build definitions globally in `.plzconfig`
under `[parse]`:

```ini
[parse]
preloadsubincludes = ///rust//build_defs:rust
```

---

## Next Steps

Explore the language-specific guides and rule references:

- **[Rust Rules Documentation](https://github.com/wcodesoft/please-rules/blob/rust/docs/rust/README.md)**:
  `rust_library`, `rust_bin`, `rust_test`, and `rust_crate` on the
  [`rust`](https://github.com/wcodesoft/please-rules/tree/rust) branch.
- **[Multi-Language Gitflow Architecture](gitflow.md)**: Details on branch
  strategy, common code syncing, and release tagging.
- **[Documentation Portal](README.md)**: Full index of rules, architecture, and
  references.
