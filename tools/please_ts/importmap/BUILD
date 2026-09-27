subinclude("///go//build_defs:go")

go_library(
    name = "importmap",
    srcs = glob(["*.go"], exclude = ["*_test.go"]),
    visibility = ["//tools/please_ts/..."],
)

go_test(
    name = "importmap_test",
    srcs = glob(["*.go"]),
)
