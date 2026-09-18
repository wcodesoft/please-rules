subinclude("///go//build_defs:go")

go_library(
    name = "toolchain",
    srcs = ["toolchain.go"],
    visibility = ["//tools/please_wit/..."],
)

go_test(
    name = "toolchain_test",
    srcs = [
        "toolchain.go",
        "toolchain_test.go",
    ],
)
