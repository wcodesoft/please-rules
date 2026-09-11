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
    revision = "v0.3.0",
)
```

As additional language rulesets (such as TypeScript/Deno) become available in
this repository, they can be declared using their respective plugin names.

---

## 2. Configuring `.plzconfig`

Enable and configure the plugin in your root `.plzconfig`:

```ini
; Please configuration file

[Plugin "rust"]
Target = //plugins:rust

[build]
passenv = PATH, HOME
```

### Environment Pass-Through (`passenv`)

For toolchains that rely on host-installed compilers or dispatchers (such as
`rustup`), passing `PATH` and `HOME` ensures that Please's sandboxed build
actions can resolve tools and their corresponding configuration files without
hardcoding paths.

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

- **[Rust Rules Documentation](../rust/README.md)**: `rust_library`, `rust_bin`,
  `rust_test`, and `rust_crate`.
- **[Documentation Portal](../README.md)**: Full index of rules, architecture,
  and references.
