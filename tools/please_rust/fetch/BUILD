subinclude("///go//build_defs:go")

go_library(
    name = "fetch",
    srcs = ["fetch.go"],
    deps = ["//tools/please_rust/toolchain"],
    visibility = ["//tools/please_rust/..."],
)

go_test(
    name = "fetch_test",
    srcs = ["fetch_test.go"],
    deps = [":fetch"],
)
