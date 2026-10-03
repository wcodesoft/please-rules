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

---

## 5. Integrating with WebAssembly Interface Types (WIT)

The Kotlin plugin works seamlessly with the `wit` plugin (`please-rules` WIT
plugin) to provide cross-language contracts, type safety, and direct WebAssembly
execution.

### Architectural Flow

```mermaid
flowchart LR
    WIT["disjoint_set.wit"] -->|kt_wit_bindgen (wit plugin)| Interface["interface DisjointSet\n(Generated Kotlin Contract)"]
    Interface -->|implements| Impl["DisjointSetImpl.kt\n(Your Business Logic)"]
    Impl -->|bundled with Bridge| Binary["kt_wasm_binary\n(kotlin plugin)"]
    Binary --> Output["structures.wasm\n(WebAssembly Library)"]
    Output --> Python["Python / TS / Go\n(Host Execution)"]
```

### Step 1: Define WIT Contract

In `definitions/structures/BUILD`:

```starlark
subinclude("///wit//build_defs:wit")

wit_library(
    name = "structures_wit",
    srcs = glob(["*.wit"]),
    package = "contract:structures",
)
```

And `definitions/structures/structures.wit`:

```wit
package contract:structures;

interface disjoint-set {
    make-set: func(x: s32);
    find: func(x: s32) -> s32;
    union: func(x: s32, y: s32);
    is-connected: func(x: s32, y: s32) -> bool;
}
```

### Step 2: Implement the Interface in Kotlin

You only write the implementation class (`DisjointSetImpl.kt`). You do **not**
need to write a manual bridge or a `main()` function:

In `src/structures/DisjointSetImpl.kt`:

```kotlin
package structures

import contract.structures.DisjointSet

class DisjointSetImpl : DisjointSet {
    private val parent = mutableMapOf<Int, Int>()

    override fun makeSet(x: Int) {
        if (!parent.containsKey(x)) parent[x] = x
    }

    override fun find(x: Int): Int? = parent[x]

    override fun union(x: Int, y: Int) {
        val rootX = find(x) ?: return
        val rootY = find(y) ?: return
        if (rootX != rootY) parent[rootX] = rootY
    }

    override fun isConnected(x: Int, y: Int): Boolean {
        val rootX = find(x) ?: return false
        val rootY = find(y) ?: return false
        return rootX == rootY
    }
}
```

### Step 3: Compile to WebAssembly with Automatic Bridge Generation

Point `kt_wasm_binary` directly to your WIT contract with `wit = "..."`. The
toolchain automatically generates the interface and `@WasmExport` bridge
delegating to `DisjointSetImpl()`:

In `src/structures/BUILD`:

```starlark
subinclude("///kotlin//build_defs:kotlin")

kt_wasm_binary(
    name = "structures_wasm",
    srcs = ["DisjointSetImpl.kt"],
    wit = "//definitions/structures:structures_wit",
    target = "wasm-js",
    main = "noCall",                        # Builds as a library wasm module
    visibility = ["PUBLIC"],
)
```

> **Note on Custom Implementations**: By default, the generated bridge
> instantiates `<Interface>Impl()` (e.g. `DisjointSetImpl()`). If your
> implementation class has a different name, specify it via
> `impl = "MyCustomClassName"`.

If you prefer to write a custom bridge manually instead of generating it from
WIT, you can omit `wit` and pass your own `Bridge.kt` annotated with
`@WasmExport` in `srcs`.

```bash
./pleasew build //src/structures:structures_wasm
```

The resulting binary `plz-out/gen/src/structures/structures_wasm.wasm` is a
standalone library WebAssembly module.

### Step 5: Call Directly in Python

```python
from wasmtime import Store, Module, Instance

store = Store()
module = Module.from_file(store.engine, "plz-out/gen/src/structures/structures_wasm.wasm")
instance = Instance(store, module, [])

exports = instance.exports(store)
exports["makeSet"](store, 1)
exports["makeSet"](store, 2)

print("Connected?", bool(exports["isConnected"](store, 1, 2)))  # False

exports["union"](store, 1, 2)
print("Connected after union?", bool(exports["isConnected"](store, 1, 2)))  # True
```

### Why There Are Zero Circular Dependencies

1. **Contract (`definitions`)**: Contains only pure interfaces
   (`babel.structures.DisjointSet`) and depends on nothing.
2. **Implementation (`src/structures`)**: Implements the contract and depends
   only on the contract.
3. **Binary (`kt_wasm_binary`)**: Compiles the implementation and export bridge
   together into `.wasm`. The dependency graph remains a strict, clean Directed
   Acyclic Graph (DAG):
   $$\text{Binary} \longrightarrow \text{Implementation} \longrightarrow \text{Contract}$$

---

## Code Coverage (`plz cover`)

### Required `[cover]` setting

Please only reports coverage for files whose extension is listed under `[cover]`
in your `.plzconfig`. The Kotlin plugin cannot set this for you; without it the
report is silently `Total coverage: No data`, or lists files of other languages
instead of yours.

```ini
[cover]
fileextension = .kt
```

If your repository mixes languages, add one `fileextension` line per extension.

### Running

```bash
./pleasew cover //src/...
```

Test targets are run with the JaCoCo agent (`JacocoAgent`) and reported with
`JacocoCli`; both are provided by `kotlin_toolchain`.

### Report output

- **Location**: Please merges the per-test results into
  `plz-out/log/coverage.xml` (Cobertura XML) and `plz-out/log/coverage.json`
  (per-file line markers plus per-file and per-directory percentages).
- **Contents**: line coverage only. Branch coverage and per-function data are
  not emitted, so consumers cannot join coverage with per-function complexity
  from this report.
- **Format**: the runner converts the JaCoCo XML report to GCOV and writes it to
  `$COVERAGE_FILE`. Only classes compiled from your own `kt_library` targets are
  reported (third-party jars bundled into the test jar are excluded), and the
  test target's own sources are not counted.

### Raw lcov export (functions and branches)

Please reduces every coverage report to per-line data. Each `kotlin_test` also
keeps the JaCoCo report converted to lcov, with the function (`FN`/`FNDA`) and
branch (`BRDA`) records that `coverage.xml` and `coverage.json` do not carry:

- **Location**: `plz-out/bin/<package>/<target name>.lcov`, for example
  `plz-out/bin/test/kotlin/branches/branches_test.lcov`. It is declared as a
  test output, so it is written next to the test jar.
- **Content**: lcov with repository-relative paths, for the project classes only
  (third-party classes bundled into the test jar are excluded). The file is
  empty unless the test ran under `plz cover`.
- **Functions**: one record per method with a source line, named
  `<package>.<Class>.<method><descriptor>`, for example
  `test.kotlin.branches.BranchesKt.both(ZZ)Z`.
- **Hit counts are 0 or 1**: JaCoCo records whether code ran, not how often.
- **Branches are anonymous**: JaCoCo reports, per line, how many branches ran
  and how many did not, without saying which. The arms of a line are numbered
  from 0 with the taken ones first, so only "N arms ran, M did not" is
  meaningful, not which arm an index refers to. The counts are JVM-level
  branches, so a short-circuit `a && b` can show more arms than the source has
  conditions. Branches of a line that never ran are written as not evaluated
  (`-`).
- **Lifetime**: the file reflects the last test run that actually executed. Read
  it right after `./pleasew cover --rerun <target>`; do not assume it is
  recreated when Please reuses a cached result.
