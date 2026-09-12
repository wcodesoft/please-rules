subinclude("///go//build_defs:go")

go_library(
    name = "testrunner",
    srcs = ["testrunner.go"],
    deps = ["//tools/please_rust/toolchain"],
    visibility = ["//tools/please_rust/..."],
)

go_test(
    name = "testrunner_test",
    srcs = ["testrunner_test.go"],
    deps = [":testrunner"],
)
