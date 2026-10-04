subinclude("///go//build_defs:go")

go_library(
    name = "unpack",
    srcs = [
        "npmcache.go",
        "resolve.go",
        "unpack.go",
    ],
    visibility = ["//tools/please_ts/..."],
    deps = [
        "//tools/please_ts/importmap",
        "//tools/please_ts/npmcache",
        "//tools/please_ts/registry",
        "//tools/please_ts/semver",
    ],
)

go_test(
    name = "unpack_test",
    srcs = [
        "npmcache.go",
        "npmcache_test.go",
        "resolve.go",
        "resolve_test.go",
        "unpack.go",
        "unpack_test.go",
    ],
    deps = [
        "//tools/please_ts/importmap",
        "//tools/please_ts/npmcache",
        "//tools/please_ts/registry",
        "//tools/please_ts/semver",
    ],
)
