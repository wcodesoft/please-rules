subinclude("///go//build_defs:go")

go_library(
    name = "npmcache",
    srcs = ["npmcache.go"],
    visibility = ["//tools/please_ts/..."],
)

go_test(
    name = "npmcache_test",
    srcs = [
        "npmcache.go",
        "npmcache_test.go",
    ],
)
