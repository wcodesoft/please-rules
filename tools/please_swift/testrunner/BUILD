subinclude("///go//build_defs:go")

go_library(
    name = "testrunner",
    srcs = ["runner.go"],
    deps = ["//tools/please_swift/linkorder"],
    visibility = ["//tools/please_swift/..."],
)

go_test(
    name = "testrunner_test",
    srcs = [
        "runner.go",
        "runner_test.go",
    ],
    deps = ["//tools/please_swift/linkorder"],
)
