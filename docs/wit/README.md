# WebAssembly Interface Types (WIT) Rules for Please

This directory contains documentation for `please-rules` WebAssembly Interface
Types (WIT) rules (`///wit`).

---

## Overview

The WIT ruleset provides support for WebAssembly Interface Types (WIT) in Please
build graphs:

- **Standalone AST Code Generation**: Pure Go lexer and AST parser inside
  `please_wit` that directly emits clean, idiomatic interface contracts for
  **Kotlin**, **Swift**, **TypeScript**, **Python**, **Rust**, **Go**, and
  **C++** with zero external dependencies (no external binaries or tools
  required).
- **Strict 3-Tier DAG Support**: Enables contract-driven multi-language
  architectures (`wit_library` $\to$ Language Interfaces $\to$ Implementation
  $\to$ WebAssembly Binaries).
- **Hermetic & Instant**: Pure Go orchestrator built directly by Please; zero
  external binary downloads, ensuring completely offline, fast builds.

---

## Rule Quick Reference

| Rule                 | Description                                                           | Toolchain Dependency                           |
| :------------------- | :-------------------------------------------------------------------- | :--------------------------------------------- |
| `wit_library`        | Packages `.wit` schema files into a consumable WIT package            | `please_wit` (pure Go, zero external binaries) |
| `kt_wit_bindgen`     | Generates clean Kotlin interfaces from WIT AST                        | `please_wit` (pure Go, zero external binaries) |
| `swift_wit_bindgen`  | Generates Swift protocols and module maps from WIT AST                | `please_wit` (pure Go, zero external binaries) |
| `ts_wit_bindgen`     | Generates TypeScript interfaces and type definitions from WIT AST     | `please_wit` (pure Go, zero external binaries) |
| `python_wit_bindgen` | Generates Python `Protocol` definitions and `.pyi` stubs from WIT AST | `please_wit` (pure Go, zero external binaries) |
| `rust_wit_bindgen`   | Generates Rust traits and data types from WIT AST                     | `please_wit` (pure Go, zero external binaries) |
| `go_wit_bindgen`     | Generates Go interfaces and struct types from WIT AST                 | `please_wit` (pure Go, zero external binaries) |
| `cc_wit_bindgen`     | Generates C++ abstract classes and headers from WIT AST               | `please_wit` (pure Go, zero external binaries) |

---

## Further Reading

- **[Rule Reference](rules.md)**: Complete parameter and API documentation for
  all WIT rules.
- **[Architecture & Design](architecture.md)**: AST parser design,
  zero-toolchain code generation, and multi-language contract workflows.
