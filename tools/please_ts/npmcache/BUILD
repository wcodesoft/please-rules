subinclude("///go//build_defs:go")

go_library(
    name = "npmcache",
    srcs = [
        "freeze.go",
        "npmcache.go",
    ],
    visibility = ["//tools/please_ts/..."],
    deps = ["//tools/please_ts/semver"],
)

go_test(
    name = "npmcache_test",
    srcs = [
        "freeze.go",
        "freeze_test.go",
        "npmcache.go",
        "npmcache_test.go",
    ],
    deps = ["//tools/please_ts/semver"],
)
