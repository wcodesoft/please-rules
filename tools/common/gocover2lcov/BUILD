subinclude("///go//build_defs:go")

go_binary(
    name = "gocover2lcov",
    srcs = ["main.go"],
    visibility = ["PUBLIC"],
    deps = ["//tools/common/lcov"],
)

go_test(
    name = "gocover2lcov_test",
    srcs = [
        "main.go",
        "main_test.go",
    ],
    deps = ["//tools/common/lcov"],
)
