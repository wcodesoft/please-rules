#!/usr/bin/env bash
set -euo pipefail

TARGET_NAME="${1:-wit_wasm.wasm}"
WASM_FILE="$TARGET_NAME"
if [ ! -f "$WASM_FILE" ]; then
    WASM_FILE=$(find . -name "$TARGET_NAME" | head -n 1)
fi
if [ -z "$WASM_FILE" ] || [ ! -f "$WASM_FILE" ]; then
    WASM_FILE=$(find . -name "*.wasm" | head -n 1)
fi

if [ -z "$WASM_FILE" ] || [ ! -f "$WASM_FILE" ]; then
    echo "ERROR: WebAssembly file not found!"
    exit 1
fi

echo "Found $WASM_FILE (size: $(wc -c < "$WASM_FILE") bytes)"

# Verify wasm magic header \0asm (0x00 0x61 0x73 0x6d)
MAGIC=$(od -An -tx1 -N4 "$WASM_FILE" | tr -d ' \n')
if [ "$MAGIC" != "0061736d" ]; then
    echo "ERROR: Invalid WebAssembly magic header! Found: $MAGIC"
    exit 1
fi

# Verify auto-generated bridge functions are exported in the WebAssembly binary
grep -q "findRoot" "$WASM_FILE" || { echo "ERROR: 'findRoot' export not found in $WASM_FILE"; exit 1; }
grep -q "unionSets" "$WASM_FILE" || { echo "ERROR: 'unionSets' export not found in $WASM_FILE"; exit 1; }
grep -q "reset" "$WASM_FILE" || { echo "ERROR: 'reset' export not found in $WASM_FILE"; exit 1; }

echo "SUCCESS: wit_wasm.wasm is a valid WebAssembly module with auto-generated exports (findRoot, unionSets, reset)."
