subinclude("///go//build_defs:go")

go_library(
    name = "compile",
    srcs = ["compile.go"],
    visibility = ["//tools/please_swift/..."],
)

go_test(
    name = "compile_test",
    srcs = [
        "compile.go",
        "compile_test.go",
    ],
)
