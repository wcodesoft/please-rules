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
    revision = "rust-v0.4.1",  # or revision = "rust"
)

# Kotlin plugin
plugin_repo(
    name = "kotlin",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "kotlin-v0.1.3",  # or revision = "kotlin"
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
    revision = "ts-v0.1.0",  # or revision = "ts"
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
RustcTool = //third_party/rust:toolchain|rustc
```

_(Where `//third_party/rust:toolchain` is defined via
`rust_toolchain(name = "toolchain", version = "1.85.0")`)._

### Kotlin

```ini
[Plugin "kotlin"]
Target = //plugins:kotlin
KotlincTool = //third_party/kotlin:toolchain|kotlinc
JavaTool = //third_party/kotlin:toolchain|java
```

### Swift

```ini
[Plugin "swift"]
Target = //plugins:swift
SwiftcTool = //third_party/swift:toolchain|swiftc
```

### TypeScript / Deno

```ini
[Plugin "ts"]
Target = //plugins:ts
DenoTool = //third_party/ts:toolchain|deno
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
```

Alternatively, you can preload build definitions globally in `.plzconfig` under
`[parse]`:

```ini
[parse]
preloadsubincludes = ///rust//build_defs:rust, ///kotlin//build_defs:kotlin, ///swift//build_defs:swift, ///ts//build_defs:ts
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
