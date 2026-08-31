subinclude("///go//build_defs:go")

go_library(
    name = "toolchain",
    srcs = ["toolchain.go"],
    visibility = ["//tools/please_rust/..."],
)

go_test(
    name = "toolchain_test",
    srcs = ["toolchain_test.go"],
    deps = [":toolchain"],
)
