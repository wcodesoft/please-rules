subinclude("///go//build_defs:go")

go_library(
    name = "compile",
    srcs = ["compile.go"],
    visibility = ["//tools/please_ts/..."],
    deps = [
        "//tools/please_ts/css",
        "//tools/please_ts/importmap",
        "//tools/please_ts/npmcache",
    ],
)

go_test(
    name = "compile_test",
    srcs = ["compile_test.go"],
    deps = [
        ":compile",
        "//tools/please_ts/css",
        "//tools/please_ts/importmap",
        "//tools/please_ts/npmcache",
    ],
)
