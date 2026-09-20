subinclude("///go//build_defs:go")

go_binary(
    name = "please_kotlin_wasm",
    srcs = [
        "compile.go",
        "main.go",
    ],
    visibility = ["PUBLIC"],
)

go_test(
    name = "please_kotlin_wasm_test",
    srcs = [
        "compile.go",
        "compile_test.go",
        "main.go",
        "main_test.go",
    ],
)
