# Getting Started with `please-rules`

This guide explains how to integrate and consume build rules from `please-rules`
in your [Please](https://please.build) project.

---

## 1. Declaring the Plugin Dependency

In your project's `plugins/BUILD` file, declare each required language plugin
using `plugin_repo`. Because `please-rules` uses a
**[Hub-and-Spoke Gitflow](gitflow.md)**, specify the respective language release
tag or branch in `revision`:

```starlark
# plugins/BUILD

# Rust plugin
plugin_repo(
    name = "rust",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "rust-v0.5.0",  # or revision = "rust"
)

# Kotlin plugin
plugin_repo(
    name = "kotlin",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "kotlin-v0.2.0",  # or revision = "kotlin"
)

# Swift plugin
plugin_repo(
    name = "swift",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "swift-v0.1.1",  # or revision = "swift"
)

# TypeScript / Deno plugin
plugin_repo(
    name = "ts",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "ts-v0.1.2",  # or revision = "ts"
)

# WebAssembly Interface Types (WIT) plugin
plugin_repo(
    name = "wit",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "wit-v0.1.0",  # or revision = "wit"
)
```

---

## 2. Configuring `.plzconfig`

Add configuration sections for each language plugin in your project's root
`.plzconfig`:

### Rust

```ini
[Plugin "rust"]
Target = //plugins:rust
RustcTool = ///rust//tools/rust_toolchain:toolchain|rustc
```

_(Where `RustcTool` points to the hermetic toolchain target or local compiler)._

### Kotlin

```ini
[Plugin "kotlin"]
Target = //plugins:kotlin
KotlincTool = ///kotlin//tools/kotlin_toolchain:toolchain|kotlinc
JavaTool = ///kotlin//tools/kotlin_toolchain:toolchain|java
```

### Swift

```ini
[Plugin "swift"]
Target = //plugins:swift
SwiftcTool = ///swift//tools/swift_toolchain:toolchain|swiftc
```

### TypeScript / Deno

```ini
[Plugin "ts"]
Target = //plugins:ts
DenoTool = ///ts//tools/ts_toolchain:toolchain|deno
```

### WebAssembly Interface Types (WIT)

```ini
[Plugin "wit"]
Target = //plugins:wit
WitBindgenTool = ///wit//tools/wit_toolchain:toolchain|wit-bindgen
WasmToolsTool = ///wit//tools/wit_toolchain:toolchain|wasm-tools
```

---

## 3. Subincluding Rules in `BUILD` Files

In any package where you want to use the rules, subinclude the rule definitions:

```starlark
# Rust targets
subinclude("///rust//build_defs:rust")

# Kotlin targets
subinclude("///kotlin//build_defs:kotlin")

# Swift targets
subinclude("///swift//build_defs:swift")

# TypeScript targets
subinclude("///ts//build_defs:ts")

# WIT targets
subinclude("///wit//build_defs:wit")
```

Alternatively, you can preload build definitions globally in `.plzconfig` under
`[parse]`:

```ini
[parse]
preloadsubincludes = ///rust//build_defs:rust, ///kotlin//build_defs:kotlin, ///swift//build_defs:swift, ///ts//build_defs:ts, ///wit//build_defs:wit
```

---

## Next Steps

Explore the language-specific guides and rule references:

- **[Rust Rules Documentation](https://github.com/wcodesoft/please-rules/blob/rust/docs/rust/README.md)**
  (`rust` branch)
- **[Kotlin Rules Documentation](https://github.com/wcodesoft/please-rules/blob/kotlin/docs/kotlin/README.md)**
  (`kotlin` branch)
- **[Swift Rules Documentation](https://github.com/wcodesoft/please-rules/blob/swift/docs/swift/README.md)**
  (`swift` branch)
- **[TypeScript Rules Documentation](https://github.com/wcodesoft/please-rules/blob/ts/docs/ts/README.md)**
  (`ts` branch)
- **[Multi-Language Gitflow Architecture](gitflow.md)**: Details on branch
  strategy, common code syncing, and release tagging.
- **[Documentation Portal](README.md)**: Full index of rules, architecture, and
  references.
