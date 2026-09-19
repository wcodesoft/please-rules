subinclude("///go//build_defs:go")

go_library(
    name = "generate",
    srcs = [
        "generate.go",
        "kotlin.go",
        "python.go",
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
        "generate.go",
        "generate_test.go",
        "kotlin.go",
        "python.go",
        "swift.go",
        "ts.go",
    ],
    deps = [
        "//tools/please_wit/ast",
    ],
)

