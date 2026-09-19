# Kotlin Usage Guide

This guide walks through configuring and using the Please Kotlin plugin in your
project.

---

## 1. Plugin Declaration

Declare the Kotlin plugin in your project's `plugins/BUILD`:

```starlark
plugin_repo(
    name = "kotlin",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "kotlin-v0.1.0",     # or revision = "kotlin"
)
```

---

## 2. Toolchain Configuration

### Setting up `third_party/kotlin/BUILD`

```starlark
subinclude("///kotlin//build_defs:kotlin")

kotlin_toolchain(
    name = "toolchain",
    version = "2.1.10",
    visibility = ["PUBLIC"],
)
```

### Configuring `.plzconfig`

Bind the plugin to the hermetic toolchain in `.plzconfig`:

```ini
[Plugin "kotlin"]
Target = //plugins:kotlin
KotlincTool = //third_party/kotlin:toolchain|kotlinc
JavaTool = //third_party/kotlin:toolchain|java
JacocoAgent = //third_party/kotlin:toolchain|jacoco-agent
JacocoCli = //third_party/kotlin:toolchain|jacoco-cli
JvmTarget = 21

[cover]
fileextension = .kt
```

---

## 3. Building and Testing

### Build targets

```bash
./pleasew build //src/...
```

### Run an application

```bash
./pleasew run //src/app:bin
```

### Run tests with JUnit results

```bash
./pleasew test //src/...
```

### Run tests with line-by-line coverage

```bash
./pleasew cover //src/...
```

---

## 4. Compiling to WebAssembly (`kt_wasm_binary`)

The Kotlin plugin supports compiling Kotlin logic into WebAssembly (`.wasm`)
library modules.

### A. Authoring Kotlin with `@WasmExport`

```kotlin
// src/math/Math.kt
package math

import kotlin.wasm.WasmExport

@WasmExport
fun add(a: Int, b: Int): Int = a + b

@WasmExport
fun multiply(a: Int, b: Int): Int = a * b
```

### B. Defining the `kt_wasm_binary` Target

```starlark
# src/math/BUILD
subinclude("///kotlin//build_defs:kotlin")

kt_wasm_binary(
    name = "math_wasm",
    srcs = ["Math.kt"],
    target = "wasm-js",  # or "wasm-wasi"
    main = "noCall",     # produces a library wasm module (reactor)
    visibility = ["PUBLIC"],
)
```

Compile the `.wasm` module:

```bash
./pleasew build //src/math:math_wasm
```

The output is available at `plz-out/gen/src/math/math_wasm.wasm`.

### C. Calling the Kotlin Wasm Library from Python

```python
from wasmtime import Store, Module, Instance

store = Store()
module = Module.from_file(store.engine, "plz-out/gen/src/math/math_wasm.wasm")
instance = Instance(store, module, [])

# Call exported Kotlin functions directly
add = instance.exports(store)["add"]
print(add(store, 10, 25))  # 35
```

### D. Calling the Kotlin Wasm Library from Node.js / Browser

```javascript
import fs from "fs";

const wasmBuffer = fs.readFileSync("plz-out/gen/src/math/math_wasm.wasm");
const { instance } = await WebAssembly.instantiate(wasmBuffer);

console.log(instance.exports.add(10, 25)); // 35
```
