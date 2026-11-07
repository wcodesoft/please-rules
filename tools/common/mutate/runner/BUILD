subinclude("///go//build_defs:go")

go_library(
    name = "runner",
    srcs = [
        "eval.go",
        "files.go",
        "options.go",
        "runner.go",
    ],
    visibility = ["//tools/..."],
    deps = [
        "//tools/common/mutation",
        "//tools/common/treesitter",
    ],
)

go_test(
    name = "runner_test",
    srcs = [
        "eval_test.go",
        "files_test.go",
        "options_test.go",
        "runner_test.go",
    ],
    deps = [
        ":runner",
        "//tools/common/mutation",
        "//tools/common/treesitter",
    ],
)
