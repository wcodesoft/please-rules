subinclude("///go//build_defs:go")

go_library(
    name = "importmap",
    srcs = glob(["*.go"], exclude = ["*_test.go"]),
    visibility = ["//tools/please_ts/..."],
    deps = ["//tools/please_ts/npmcache"],
)

go_test(
    name = "importmap_test",
    srcs = glob(["*.go"]),
    deps = ["//tools/please_ts/npmcache"],
)
