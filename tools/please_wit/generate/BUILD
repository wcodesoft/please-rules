subinclude("///go//build_defs:go")

go_library(
    name = "generate",
    srcs = [
        "cpp.go",
        "generate.go",
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
        "cpp.go",
        "generate.go",
        "generate_test.go",
        "golang.go",
        "kotlin.go",
        "python.go",
        "rust.go",
        "swift.go",
        "ts.go",
    ],
    deps = [
        "//tools/please_wit/ast",
    ],
)

