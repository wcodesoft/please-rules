subinclude("///go//build_defs:go")

go_library(
    name = "bundle",
    srcs = [
        "bundle.go",
        "denobundle.go",
    ],
    visibility = ["//tools/please_ts/..."],
    deps = [
        "//tools/please_ts/css",
        "//tools/please_ts/importmap",
        "//tools/please_ts/npmcache",
    ],
)

go_test(
    name = "bundle_test",
    srcs = [
        "bundle_test.go",
        "denobundle_test.go",
    ],
    deps = [
        ":bundle",
        "//tools/please_ts/css",
        "//tools/please_ts/importmap",
        "//tools/please_ts/npmcache",
    ],
)
