subinclude("///go//build_defs:go")

go_library(
    name = "download",
    srcs = ["download.go"],
    visibility = ["//tools/please_rust/..."],
)

go_test(
    name = "download_test",
    srcs = ["download_test.go"],
    deps = [":download"],
)
