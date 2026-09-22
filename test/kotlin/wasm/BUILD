subinclude("//build_defs/kotlin:kotlin")

kt_wasm_binary(
    name = "math_wasm",
    srcs = ["math.kt"],
    target = "wasm-js",
    main = "noCall",
    visibility = ["PUBLIC"],
)

gentest(
    name = "math_wasm_test",
    data = [
        ":math_wasm",
        "verify_wasm.sh",
    ],
    test_cmd = "$(location verify_wasm.sh)",
    no_test_output = True,
)

kt_wasm_binary(
    name = "wit_wasm",
    srcs = ["DisjointSetImpl.kt"],
    wit = "structures.wit",
    target = "wasm-js",
    main = "noCall",
    visibility = ["PUBLIC"],
)

gentest(
    name = "wit_wasm_test",
    data = [
        ":wit_wasm",
        "verify_wit_wasm.sh",
    ],
    test_cmd = "$(location verify_wit_wasm.sh)",
    no_test_output = True,
)

kt_wasm_binary(
    name = "disjoint_set_wasm",
    srcs = ["DisjointSet.kt"],
    wit = "structures.wit",
    target = "wasm-js",
    main = "noCall",
    visibility = ["PUBLIC"],
)

gentest(
    name = "disjoint_set_wasm_test",
    data = [
        ":disjoint_set_wasm",
        "verify_wit_wasm.sh",
    ],
    test_cmd = "$(location verify_wit_wasm.sh) $(location :disjoint_set_wasm)",
    no_test_output = True,
)

kt_wasm_binary(
    name = "disjoint_set_wasi",
    srcs = ["DisjointSet.kt"],
    wit = "structures.wit",
    target = "wasm-wasi",
    main = "noCall",
    visibility = ["PUBLIC"],
)

gentest(
    name = "disjoint_set_wasi_test",
    data = [
        ":disjoint_set_wasi",
        "verify_wit_wasm.sh",
    ],
    test_cmd = "$(location verify_wit_wasm.sh) $(location :disjoint_set_wasi)",
    no_test_output = True,
)
