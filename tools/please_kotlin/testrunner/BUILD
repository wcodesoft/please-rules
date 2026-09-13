subinclude("///go//build_defs:go")

go_library(
    name = "testrunner",
    srcs = [
        "coverage.go",
        "junit.go",
        "runner.go",
    ],
    deps = [
        "//tools/please_kotlin/compile",
        "//tools/please_kotlin/toolchain",
    ],
    visibility = ["//tools/please_kotlin/..."],
)

go_test(
    name = "testrunner_test",
    srcs = [
        "coverage_test.go",
        "junit_test.go",
        "runner_test.go",
    ],
    deps = [
        ":testrunner",
    ],
)
