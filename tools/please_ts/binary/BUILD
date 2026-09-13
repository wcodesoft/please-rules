subinclude("///go//build_defs:go")

go_library(
    name = "binary",
    srcs = ["binary.go"],
    visibility = ["//tools/please_ts/..."],
    deps = ["//tools/please_ts/importmap"],
)

go_test(
    name = "binary_test",
    srcs = [
        "binary.go",
        "binary_test.go",
    ],
    deps = ["//tools/please_ts/importmap"],
)
