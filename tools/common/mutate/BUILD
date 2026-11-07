subinclude("///go//build_defs:go")

go_binary(
    name = "mutate",
    srcs = [
        "main.go",
        "runner.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//tools/common/mutation",
        "//tools/common/treesitter",
    ],
)

go_test(
    name = "mutate_test",
    srcs = [
        "main_test.go",
        "runner_test.go",
    ],
    deps = [
        ":mutate",
        "//tools/common/mutation",
        "//tools/common/treesitter",
    ],
)
