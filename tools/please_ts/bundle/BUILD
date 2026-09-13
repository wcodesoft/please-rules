subinclude("///go//build_defs:go")

go_library(
    name = "bundle",
    srcs = ["bundle.go"],
    visibility = ["//tools/please_ts/..."],
    deps = ["//tools/please_ts/importmap"],
)

go_test(
    name = "bundle_test",
    srcs = [
        "bundle.go",
        "bundle_test.go",
    ],
    deps = ["//tools/please_ts/importmap"],
)
