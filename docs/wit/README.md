# WebAssembly Interface Types (WIT) Rules for Please

This directory contains documentation for `please-rules` WebAssembly Interface
Types (WIT) rules (`///wit`).

---

## Overview

The WIT ruleset provides support for WebAssembly Interface Types (WIT) in Please
build graphs:

- **Standalone AST Code Generation**: Pure Go lexer and AST parser inside
  `please_wit` that directly emits clean, idiomatic interface contracts for
  **Kotlin**, **Swift**, **TypeScript**, and **Python** without downloading or
  requiring external `wit-bindgen` or `wasm-tools` binaries.
- **Hermetic Toolchain for Rust/Go/C++**: Seamlessly integrates official
  `wit-bindgen` for languages with native upstream backend support.
- **Strict 3-Tier DAG Support**: Enables contract-driven multi-language
  architectures (`wit_library` $\to$ Language Interfaces $\to$ Implementation
  $\to$ WebAssembly Binaries).

---

## Rule Quick Reference

| Rule                 | Description                                                           | Toolchain Dependency                           |
| :------------------- | :-------------------------------------------------------------------- | :--------------------------------------------- |
| `wit_library`        | Packages `.wit` schema files into a consumable WIT package            | `please_wit` (pure Go)                         |
| `kt_wit_bindgen`     | Generates clean Kotlin interfaces from WIT AST                        | `please_wit` (pure Go, zero external binaries) |
| `swift_wit_bindgen`  | Generates Swift protocols and module maps from WIT AST                | `please_wit` (pure Go, zero external binaries) |
| `ts_wit_bindgen`     | Generates TypeScript interfaces and type definitions from WIT AST     | `please_wit` (pure Go, zero external binaries) |
| `python_wit_bindgen` | Generates Python `Protocol` definitions and `.pyi` stubs from WIT AST | `please_wit` (pure Go, zero external binaries) |
| `rust_wit_bindgen`   | Generates Rust Guest traits and bindings                              | `wit-bindgen`                                  |
| `go_wit_bindgen`     | Generates Go interfaces and C bindings                                | `wit-bindgen`                                  |
| `cc_wit_bindgen`     | Generates C++ headers and bindings                                    | `wit-bindgen`                                  |
| `wit_toolchain`      | Optional hermetic toolchain for `wit-bindgen` & `wasm-tools`          | External tarballs                              |

---

## Further Reading

- **[Rule Reference](rules.md)**: Complete parameter and API documentation for
  all WIT rules.
- **[Architecture & Design](architecture.md)**: AST parser design,
  zero-toolchain code generation, and multi-language contract workflows.
