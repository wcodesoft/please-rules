# Changelog

All notable changes to the WIT build rules plugin will be documented in this
file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-09-18

### Added

- `wit_library`: Packages one or more `.wit` source files into a WIT interface
  package for use as input to binding generators.
- `wit_toolchain`: Hermetic toolchain rule downloading precompiled `wit-bindgen`
  (v0.62.0) and `wasm-tools` (v1.259.0) binaries for Linux (amd64, arm64) and
  macOS (amd64, arm64).
- `_wit_bindgen` (private): Core generator rule delegating to the `please_wit`
  Go orchestrator CLI for hermetic, language-agnostic binding generation.
- Language binding wrappers, each delegating to `_wit_bindgen`:
  - `swift_wit_bindgen`: Generates Swift C-interop bridging bindings.
  - `kt_wit_bindgen`: Generates Kotlin WASI-compatible C bindings.
  - `rust_wit_bindgen`: Generates Rust guest traits and bindings.
  - `go_wit_bindgen`: Generates Go interface bindings.
  - `ts_wit_bindgen`: Generates TypeScript definition bindings.
  - `cc_wit_bindgen`: Generates C++ header and source bindings.
  - `python_wit_bindgen` / `py_wit_bindgen`: Generates Python type stubs and
    bindings.
- `please_wit` Go CLI orchestrator providing `toolchain`, `package`, and
  `generate` subcommands to drive hermetic WIT processing.
- Unified `worlds: list = []` parameter across all binding rules: omit for
  full-package generation, or supply specific world names as a list.
