subinclude("///go//build_defs:go")

go_library(
    name = "testrunner",
    srcs = ["runner.go"],
    visibility = ["//tools/please_ts/..."],
    deps = ["//tools/please_ts/importmap"],
)

go_test(
    name = "runner_test",
    srcs = [
        "runner.go",
        "runner_test.go",
    ],
    deps = ["//tools/please_ts/importmap"],
)
