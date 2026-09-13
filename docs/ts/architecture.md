# TypeScript Rules Architecture

This document describes the architectural design of `please-rules`'s TypeScript
toolchain, compiler orchestration, and testing pipeline.

---

## 1. Hermetic Toolchain (`ts_toolchain`)

The `ts_toolchain` rule downloads standalone Deno release archives and packages
a self-contained compiler and runtime:

```txt
plz-out/bin/tools/ts_toolchain/toolchain/
└── bin/
    └── deno       # Hermetic standalone Deno executable (v2.2.3)
```

### Hermeticity Mechanism

Compilations and tests explicitly set `DENO_DIR="$TMP_DIR/.deno_cache"` and run
with `--no-remote`. Deno never connects to the internet or modifies host caches
during build execution.

---

## 2. CLI Helper (`tools/please_ts`)

Written in Go and compiled with Please's hermetic Go toolchain, `please_ts`
orchestrates build actions:

- **`compile`**:
  - Validates source files and dependency outputs.
  - Generates a sandbox-local ephemeral `import_map.json` strictly containing
    declared `deps`.
  - Invokes `deno check --no-remote --import-map .import_map.json`.
  - Emits library output and `ts_metadata.json` for upstream consumers.
- **`binary`**:
  - Invokes
    `deno compile --no-remote --import-map .import_map.json -o <out> <flags...> <main>`.
  - Produces a single, self-contained native executable embedding the Deno
    runtime and bundled code.
- **`bundle`**:
  - Assembles sources and dependencies into a single distribution asset (`.js`),
    supporting `esm` and `iife` formats, minification, and source maps.
- **`testrunner`**:
  - Runs tests strictly offline using Deno's native runner or Vitest.
  - Emits standard JUnit XML (`test.results`) seamlessly integrated with Please.
  - Emits lcov coverage data when `--coverage` is enabled.
- **`unpack`**:
  - Unpacks third-party npm package tarballs and inspects `package.json` to
    generate `ts_module.json` metadata.

---

## 3. Ephemeral Import Maps & Boundary Enforcement

Every compilation and test target receives an ephemeral `import_map.json`
synthesized solely from its declared `deps`:

```json
{
  "imports": {
    "@domain/calculator": "./test/ts/lib/calculator/calculator.ts",
    "@domain/calculator/": "./test/ts/lib/calculator/",
    "clsx": "./test/ts/npm/clsx/dist/clsx.mjs"
  }
}
```

If a source file attempts to import any undeclared module, compilation fails
immediately because Deno runs with `--no-remote` and the module specifier is
absent from the target's import map.
