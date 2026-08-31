subinclude("///go//build_defs:go")

go_library(
    name = "compile",
    srcs = ["compile.go"],
    deps = ["//tools/please_rust/toolchain"],
    visibility = ["//tools/please_rust/..."],
)

go_test(
    name = "compile_test",
    srcs = ["compile_test.go"],
    deps = [":compile"],
)
