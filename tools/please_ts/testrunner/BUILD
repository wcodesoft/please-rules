subinclude("///go//build_defs:go")

go_library(
    name = "testrunner",
    srcs = [
        "cdp.go",
        "runner.go",
    ],
    visibility = ["//tools/please_ts/..."],
    deps = [
        "//tools/please_ts/bundle",
        "//tools/please_ts/importmap",
    ],
)

go_test(
    name = "runner_test",
    srcs = [
        "cdp.go",
        "runner.go",
        "runner_test.go",
    ],
    deps = [
        "//tools/please_ts/bundle",
        "//tools/please_ts/importmap",
    ],
)
