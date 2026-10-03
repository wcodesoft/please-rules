subinclude("///go//build_defs:go")

go_library(
    name = "testrunner",
    srcs = [
        "lcov_export.go",
        "runner.go",
    ],
    deps = [
        "//tools/common/lcov",
        "//tools/please_swift/linkorder",
    ],
    visibility = ["//tools/please_swift/..."],
)

go_test(
    name = "testrunner_test",
    srcs = [
        "lcov_export.go",
        "lcov_export_test.go",
        "runner.go",
        "runner_test.go",
    ],
    deps = [
        "//tools/common/lcov",
        "//tools/please_swift/linkorder",
    ],
)
