# Kotlin Rule Reference

This document provides the API reference for the Please Kotlin build
definitions.

---

## `kotlin_toolchain`

Assembles a hermetic Kotlin compiler, OpenJDK runtime, and JaCoCo coverage
sysroot.

```starlark
kotlin_toolchain(
    name = "toolchain",
    version = "2.1.10",
    jdk_version = "21",
    visibility = ["PUBLIC"],
)
```

### Arguments

| Name              | Type  | Default       | Description                                         |
| :---------------- | :---- | :------------ | :-------------------------------------------------- |
| `name`            | `str` | `"toolchain"` | Name of the toolchain target.                       |
| `version`         | `str` | `"2.4.10"`    | Version of the Kotlin compiler.                     |
| `jdk_version`     | `str` | `"21"`        | Major version of OpenJDK to bundle.                 |
| `kotlinc_url`     | `str` | `""`          | Optional override URL for Kotlin compiler archive.  |
| `kotlinc_hash`    | `str` | `""`          | Optional override SHA-256 hash for Kotlin compiler. |
| `jdk_url`         | `str` | `""`          | Optional override URL for JDK archive.              |
| `jdk_hash`        | `str` | `""`          | Optional override SHA-256 hash for JDK.             |
| `jacoco_version`  | `str` | `"0.8.12"`    | Version of JaCoCo distribution.                     |
| `target_platform` | `str` | `""`          | Optional target platform override.                  |

### Entry Points

- `:toolchain|kotlinc`: Path to hermetic `kotlinc` binary.
- `:toolchain|kotlinc-wasm`: Path to hermetic `kotlinc-wasm` compiler.
- `:toolchain|kotlin`: Path to hermetic `kotlin` runner.
- `:toolchain|java`: Path to hermetic `java` binary.
- `:toolchain|jar`: Path to hermetic `jar` packager.
- `:toolchain|jacoco-agent`: Path to `jacocoagent.jar`.
- `:toolchain|jacoco-cli`: Path to `jacococli.jar`.

---

## `kotlin_library`

Compiles Kotlin source files into a reusable `.jar` file.

```starlark
kotlin_library(
    name = "math",
    srcs = ["Math.kt"],
    deps = ["//third_party/kotlin:stdlib"],
    jvm_target = "21",
)
```

### Arguments

| Name         | Type   | Default  | Description                                       |
| :----------- | :----- | :------- | :------------------------------------------------ |
| `name`       | `str`  | Required | Name of the library target.                       |
| `srcs`       | `list` | Required | List of `.kt` source files.                       |
| `deps`       | `list` | `[]`     | List of dependency targets producing JAR files.   |
| `jvm_target` | `str`  | `""`     | Target JVM bytecode version (defaults to config). |
| `flags`      | `list` | `[]`     | Additional flags to pass directly to `kotlinc`.   |

---

## `kotlin_binary`

Produces an executable application JAR and launcher script.

```starlark
kotlin_binary(
    name = "server",
    srcs = ["Main.kt"],
    deps = [":math"],
    main_class = "com.example.MainKt",
)
```

### Arguments

| Name         | Type   | Default  | Description                                                         |
| :----------- | :----- | :------- | :------------------------------------------------------------------ |
| `name`       | `str`  | Required | Name of the binary target.                                          |
| `srcs`       | `list` | `[]`     | Source files to compile into the application JAR.                   |
| `deps`       | `list` | `[]`     | Dependency libraries.                                               |
| `main_class` | `str`  | `""`     | Fully qualified main class name (auto-derived from package & file). |
| `jvm_target` | `str`  | `""`     | Target JVM bytecode version.                                        |

---

## `kt_wasm_binary` (alias `kotlin_wasm_binary`)

Compiles Kotlin source files into a standalone WebAssembly (`.wasm`) binary
module.

```starlark
kt_wasm_binary(
    name = "math_wasm",
    srcs = ["Math.kt"],
    target = "wasm-js",   # or "wasm-wasi"
    main = "noCall",      # creates a library wasm module (reactor)
)
```

### Library Wasm vs. Command Wasm

`kt_wasm_binary` supports two execution models controlled by the `main`
argument:

- **Library Wasm Module (`main = "noCall"`, default)**: Compiles the `.wasm`
  file as a **reactor/library module**. It exports functions marked with
  `@WasmExport` without calling a main routine. External runtimes (Python via
  `wasmtime`, Node.js/browsers via `WebAssembly.instantiate`, Go via `wazero`)
  can invoke exported functions directly.
- **Executable Wasm Module (`main = "call"`)**: Compiles as an executable
  command module that automatically executes Kotlin's `fun main()` upon
  instantiation.

### Arguments

| Name         | Type   | Default     | Description                                                                                   |
| :----------- | :----- | :---------- | :-------------------------------------------------------------------------------------------- |
| `name`       | `str`  | Required    | Name of the target; outputs `<name>.wasm`.                                                    |
| `srcs`       | `list` | `[]`        | Kotlin source files (`.kt`) or directories containing `.kt` files (e.g. implementation).      |
| `wit`        | `str`  | `""`        | Optional path to a `.wit` file or directory for automatic interface and bridge generation.    |
| `impl`       | `str`  | `""`        | Optional implementation class name to instantiate in the bridge (default: `<Interface>Impl`). |
| `deps`       | `list` | `[]`        | Dependency klib targets or directories containing `.klib` files.                              |
| `target`     | `str`  | `"wasm-js"` | WebAssembly compilation target: `"wasm-js"` or `"wasm-wasi"`.                                 |
| `main`       | `str`  | `"noCall"`  | Execution mode: `"noCall"` (library/reactor) or `"call"` (command with main).                 |
| `flags`      | `list` | `[]`        | Additional flags passed directly to `kotlinc-wasm`.                                           |
| `visibility` | `list` | `None`      | Target visibility.                                                                            |
| `labels`     | `list` | `None`      | Rule labels (defaults to `["kotlin", "wasm", "bin"]`).                                        |

---

## `kt_wasm_library` (alias `kotlin_wasm_library`)

Compiles Kotlin source files into a reusable WebAssembly library (`.klib`)
package. Downstream `kt_wasm_binary` or `kt_wasm_library` targets can depend on
it via `deps`. Please automatically discovers and links all transitive `.klib`
dependencies.

```starlark
kt_wasm_library(
    name = "math_lib",
    srcs = ["Math.kt"],
    target = "wasm-wasi",   # or "wasm-js"
    visibility = ["PUBLIC"],
)
```

### Arguments

| Name         | Type   | Default     | Description                                                                        |
| :----------- | :----- | :---------- | :--------------------------------------------------------------------------------- |
| `name`       | `str`  | Required    | Name of the target; outputs `<name>.klib`.                                         |
| `srcs`       | `list` | `[]`        | Kotlin source files (`.kt`) or directories containing `.kt` files.                 |
| `deps`       | `list` | `[]`        | Upstream `kt_wasm_library` targets or directories containing `.klib` dependencies. |
| `target`     | `str`  | `"wasm-js"` | WebAssembly compilation target: `"wasm-js"` or `"wasm-wasi"`.                      |
| `flags`      | `list` | `[]`        | Additional flags passed directly to `kotlinc-wasm`.                                |
| `visibility` | `list` | `None`      | Target visibility.                                                                 |
| `labels`     | `list` | `None`      | Rule labels (defaults to `["kotlin", "wasm", "lib"]`).                             |

---

## `kotlin_test`

Runs a Kotlin test suite using JUnit 5 Platform ConsoleLauncher, generating
JUnit XML reports and JaCoCo coverage metrics.

```starlark
maven_jar(
    name = "junit_standalone",
    id = "org.junit.platform:junit-platform-console-standalone:1.11.4",
)

kotlin_test(
    name = "math_test",
    srcs = ["MathTest.kt"],
    test_class = "com.example.MathTest",
    deps = [
        ":math",
        ":junit_standalone",
    ],
)
```

### Arguments

| Name           | Type   | Default  | Description                                                     |
| :------------- | :----- | :------- | :-------------------------------------------------------------- |
| `name`         | `str`  | Required | Name of the test target.                                        |
| `srcs`         | `list` | Required | Test source files.                                              |
| `test_class`   | `str`  | `""`     | Fully qualified JUnit test class name (mandatory / derived).    |
| `deps`         | `list` | `[]`     | Dependency libraries under test, including user's JUnit runner. |
| `test_package` | `str`  | `""`     | Explicit package override if different from directory layout.   |
| `jvm_target`   | `str`  | `""`     | Target JVM bytecode version.                                    |

---

## `kotlin_jvm_import`

Imports an external precompiled JAR into the Please build graph.

```starlark
kotlin_jvm_import(
    name = "guava",
    jar = "guava-33.0.0.jar",
)
```

---

## `maven_jar`

Downloads and integrates an external dependency JAR from Maven Central or a
custom repository.

```starlark
maven_jar(
    name = "annotations",
    id = "org.jetbrains:annotations:26.0.1",
    sha256 = "2037be378980d3ba9333e97955f3b2cde392aa124d04ca73ce2eee6657199297",
)
```

### Arguments

| Name         | Type   | Default                       | Description                                              |
| :----------- | :----- | :---------------------------- | :------------------------------------------------------- |
| `name`       | `str`  | Required                      | Name of the target.                                      |
| `id`         | `str`  | `""`                          | Coordinate in `group:artifact:version` format.           |
| `group`      | `str`  | `""`                          | Maven group ID (alternative to `id`).                    |
| `artifact`   | `str`  | `""`                          | Maven artifact ID (alternative to `id`).                 |
| `version`    | `str`  | `""`                          | Artifact version (alternative to `id`).                  |
| `sha256`     | `str`  | `""`                          | Expected SHA-256 integrity hash (optional, recommended). |
| `sha1`       | `str`  | `""`                          | Expected SHA-1 integrity hash (optional).                |
| `repository` | `str`  | `https://repo1.maven.org/...` | Repository base URL (defaults to Maven Central).         |
| `deps`       | `list` | `[]`                          | Optional transitive dependencies.                        |
