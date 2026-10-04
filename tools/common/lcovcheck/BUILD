subinclude("///go//build_defs:go")

go_binary(
    name = "lcovcheck",
    srcs = ["main.go"],
    visibility = ["PUBLIC"],
    deps = ["//tools/common/lcov"],
)

go_test(
    name = "lcovcheck_test",
    srcs = [
        "main.go",
        "main_test.go",
    ],
    deps = ["//tools/common/lcov"],
)
