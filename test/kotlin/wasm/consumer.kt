package test.wasm

import kotlin.wasm.WasmExport

@OptIn(kotlin.wasm.ExperimentalWasmInterop::class)
@WasmExport
fun compute(a: Int, b: Int): Int {
    return add(a, b) + multiply(a, b)
}
