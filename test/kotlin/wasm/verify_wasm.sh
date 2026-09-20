#!/usr/bin/env bash
set -euo pipefail

WASM_FILE="test/kotlin/wasm/math_wasm.wasm"
if [ ! -f "$WASM_FILE" ]; then
    WASM_FILE=$(find . -name "math_wasm.wasm" | head -n 1)
fi

if [ ! -f "$WASM_FILE" ]; then
    echo "ERROR: math_wasm.wasm not found!"
    exit 1
fi

echo "Found $WASM_FILE (size: $(wc -c < "$WASM_FILE") bytes)"

# Verify wasm magic header \0asm (0x00 0x61 0x73 0x6d)
MAGIC=$(od -An -tx1 -N4 "$WASM_FILE" | tr -d ' \n')
if [ "$MAGIC" != "0061736d" ]; then
    echo "ERROR: Invalid WebAssembly magic header! Found: $MAGIC"
    exit 1
fi

# Verify exported symbols exist in the binary
grep -q "add" "$WASM_FILE" || { echo "ERROR: 'add' export not found in $WASM_FILE"; exit 1; }
grep -q "multiply" "$WASM_FILE" || { echo "ERROR: 'multiply' export not found in $WASM_FILE"; exit 1; }

echo "SUCCESS: math_wasm.wasm is a valid WebAssembly module exporting add and multiply."
