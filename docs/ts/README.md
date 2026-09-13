# TypeScript & Deno Rules for Please

This directory contains the first-class, hermetic TypeScript and Deno build
rules for [Please](https://please.build).

---

## Highlights

- **100% Hermetic**: Uses a standalone, pinned Deno toolchain (`v2.2.3`). No
  host-installed Node.js, npm, or Deno required.
- **Zero `node_modules`**: No committed or generated `node_modules/` directories
  in the workspace.
- **Zero Foreign Lockfiles**: Dependencies and module graphs are declared purely
  as native Please build definitions.
- **Strict Boundary Enforcement**: Builds execute strictly offline with
  `--no-remote`. Targets can only import their declared `deps` via target-local
  ephemeral import maps.
- **Dual-Mode Orchestration**: Uses the `please_ts` helper tool for import map
  synthesis, type checking, bundling, testing, and npm tarball unpacking.

---

## Quick Start

### 1. Configure the Plugin

In `plugins/BUILD`:

```starlark
plugin_repo(
    name = "ts",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "ts-v0.1.0",
)
```

In `.plzconfig`:

```ini
[Plugin "ts"]
Target = //plugins:ts
DenoTool = //tools/ts_toolchain:toolchain|deno
```

### 2. Define Targets

In your `BUILD` file:

```starlark
subinclude("///ts//build_defs:ts")

# 1. Declare third-party npm dependencies
ts_module(
    name = "clsx",
    package = "clsx",
    version = "2.1.1",
    hashes = ["6320104b491ca645120e8f99589ce663ff1fb337ba80066719a011bd95320303"],
)

# 2. Compile a modular library
ts_library(
    name = "calculator",
    srcs = ["calculator.ts"],
    module_name = "@domain/calculator",
    visibility = ["PUBLIC"],
)

# 3. Compile a standalone native executable
ts_binary(
    name = "app",
    main = "main.ts",
    deps = [":calculator"],
)

# 4. Bundle a web distribution asset
ts_bundle(
    name = "app_bundle",
    main = "main.ts",
    deps = [":calculator"],
    format = "esm",
)

# 5. Run hermetic unit tests
ts_test(
    name = "calculator_test",
    srcs = ["calculator_test.ts"],
    deps = [":calculator"],
)
```

---

## Documentation

- [Architecture Guide](architecture.md): Deep-dive into toolchain sandboxing and
  ephemeral import maps.
- [Rules Reference](rules.md): Detailed parameter reference for all build
  definitions.
- [Usage & Examples](usage.md): Practical usage examples, testing patterns, and
  bundling workflows.
