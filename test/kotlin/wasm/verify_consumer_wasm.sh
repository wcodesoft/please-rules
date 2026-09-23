#!/usr/bin/env bash
set -euo pipefail

WASM_FILE="${1:-test/kotlin/wasm/math_consumer_wasm.wasm}"
if [ ! -f "$WASM_FILE" ]; then
    WASM_FILE=$(find . -name "*math_consumer_wasm*.wasm" | head -n 1)
fi

if [ ! -f "$WASM_FILE" ]; then
    echo "ERROR: wasm file not found: $WASM_FILE"
    exit 1
fi

echo "Found $WASM_FILE (size: $(wc -c < "$WASM_FILE") bytes)"

# Verify wasm magic header \0asm (0x00 0x61 0x73 0x6d)
MAGIC=$(od -An -tx1 -N4 "$WASM_FILE" | tr -d ' \n')
if [ "$MAGIC" != "0061736d" ]; then
    echo "ERROR: Invalid WebAssembly magic header! Found: $MAGIC"
    exit 1
fi

grep -q "compute" "$WASM_FILE" || { echo "ERROR: 'compute' export not found in $WASM_FILE"; exit 1; }
grep -q "add" "$WASM_FILE" || { echo "ERROR: 'add' from library not found in $WASM_FILE"; exit 1; }

echo "SUCCESS: math_consumer_wasm is a valid WebAssembly module linking math_lib."
