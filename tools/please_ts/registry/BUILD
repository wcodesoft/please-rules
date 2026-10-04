subinclude("///go//build_defs:go")

go_library(
    name = "registry",
    srcs = ["registry.go"],
    visibility = ["//tools/please_ts/..."],
    deps = ["//tools/please_ts/semver"],
)

go_test(
    name = "registry_test",
    srcs = [
        "registry.go",
        "registry_test.go",
    ],
    deps = ["//tools/please_ts/semver"],
)
