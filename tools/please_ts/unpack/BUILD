subinclude("///go//build_defs:go")

go_library(
    name = "unpack",
    srcs = [
        "npmcache.go",
        "unpack.go",
    ],
    visibility = ["//tools/please_ts/..."],
    deps = [
        "//tools/please_ts/importmap",
        "//tools/please_ts/npmcache",
    ],
)

go_test(
    name = "unpack_test",
    srcs = [
        "npmcache.go",
        "npmcache_test.go",
        "unpack.go",
        "unpack_test.go",
    ],
    deps = [
        "//tools/please_ts/importmap",
        "//tools/please_ts/npmcache",
    ],
)
