subinclude("///go//build_defs:go")

go_library(
    name = "generate",
    srcs = [
        "cpp.go",
        "generate.go",
        "generator.go",
        "golang.go",
        "kotlin.go",
        "python.go",
        "rust.go",
        "swift.go",
        "ts.go",
    ],
    visibility = ["//tools/please_wit/..."],
    deps = [
        "//tools/please_wit/ast",
    ],
)

go_test(
    name = "generate_test",
    srcs = [
        "cpp_test.go",
        "generate_test.go",
        "golang_test.go",
        "kotlin_test.go",
        "python_test.go",
        "rust_test.go",
        "swift_test.go",
        "ts_test.go",
    ],
    deps = [
        ":generate",
        "//tools/please_wit/ast",
    ],
)

