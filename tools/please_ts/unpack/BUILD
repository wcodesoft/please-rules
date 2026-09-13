subinclude("///go//build_defs:go")

go_library(
    name = "unpack",
    srcs = ["unpack.go"],
    visibility = ["//tools/please_ts/..."],
    deps = ["//tools/please_ts/importmap"],
)

go_test(
    name = "unpack_test",
    srcs = [
        "unpack.go",
        "unpack_test.go",
    ],
    deps = ["//tools/please_ts/importmap"],
)
