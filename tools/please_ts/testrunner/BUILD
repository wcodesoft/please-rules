subinclude("///go//build_defs:go")

go_library(
    name = "testrunner",
    srcs = glob(["*.go"], exclude = ["*_test.go"]),
    visibility = ["//tools/please_ts/..."],
    deps = [
        "//tools/common/lcov",
        "//tools/please_ts/bundle",
        "//tools/please_ts/importmap",
        "//tools/please_ts/npmcache",
    ],
)

go_test(
    name = "runner_test",
    srcs = glob(["*.go"]),
    deps = [
        "//tools/common/lcov",
        "//tools/please_ts/bundle",
        "//tools/please_ts/importmap",
        "//tools/please_ts/npmcache",
    ],
)
