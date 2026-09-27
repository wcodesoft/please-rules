subinclude("///go//build_defs:go")

go_library(
    name = "testrunner",
    srcs = glob(["*.go"], exclude = ["*_test.go"]),
    visibility = ["//tools/please_ts/..."],
    deps = [
        "//tools/please_ts/bundle",
        "//tools/please_ts/importmap",
    ],
)

go_test(
    name = "runner_test",
    srcs = glob(["*.go"]),
    deps = [
        "//tools/please_ts/bundle",
        "//tools/please_ts/importmap",
    ],
)
