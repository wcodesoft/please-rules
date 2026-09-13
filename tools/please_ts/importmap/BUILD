subinclude("///go//build_defs:go")

go_library(
    name = "importmap",
    srcs = ["importmap.go"],
    visibility = ["//tools/please_ts/..."],
)

go_test(
    name = "importmap_test",
    srcs = [
        "importmap.go",
        "importmap_test.go",
    ],
)
