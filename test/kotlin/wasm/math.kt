package test.wasm

import kotlin.wasm.WasmExport

@OptIn(kotlin.wasm.ExperimentalWasmInterop::class)
@WasmExport
fun add(a: Int, b: Int): Int {
    return a + b
}

@OptIn(kotlin.wasm.ExperimentalWasmInterop::class)
@WasmExport
fun multiply(a: Int, b: Int): Int {
    return a * b
}
