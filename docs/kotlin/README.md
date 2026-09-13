# Please Kotlin Rules (`please-rules`)

Modern, 100% hermetic build rules and toolchain management for Kotlin on the
[Please](https://please.build) build system.

---

## Features

- **100% Hermetic**: Self-contained `kotlin_toolchain` downloading `kotlinc`
  2.1.10, Temurin OpenJDK 21, and JaCoCo 0.8.12. No host compiler or Java
  installation required.
- **Reproducible & Verified**: All toolchain artifacts verified against
  cryptographic SHA-256 hashes.
- **Cross-Platform**: Full support for Linux (`x86_64`, `aarch64`) and macOS
  (`aarch64`, `x86_64`).
- **First-Class Rules**: `kotlin_library`, `kotlin_binary`, `kotlin_test`, and
  `kotlin_jvm_import`.
- **Automated Testing & Telemetry**: Generates standard `test.results` JUnit XML
  compatible with Please test logs and Veritas.
- **Code Coverage (`plz cover`)**: Automated JaCoCo agent instrumentation,
  report parsing, and Please-compatible GCOV/LCOV generation.

---

## Documentation Index

- **[Architecture & Design](architecture.md)**: Deep dive into the sysroot
  layout, `please_kotlin` CLI orchestrator, and JaCoCo coverage pipeline.
- **[Rule Reference](rules.md)**: Complete API reference for all Kotlin build
  rules and configuration options.
- **[Usage Guide](usage.md)**: Step-by-step guides for `.plzconfig` setup,
  third-party dependencies, and testing.
