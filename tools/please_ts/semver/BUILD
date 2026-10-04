subinclude("///go//build_defs:go")

go_library(
    name = "semver",
    srcs = ["semver.go"],
    visibility = ["//tools/please_ts/..."],
)

go_test(
    name = "semver_test",
    srcs = [
        "semver.go",
        "semver_test.go",
    ],
)
