subinclude("///go//build_defs:go")

go_binary(
    name = "mutate",
    srcs = [
        "main.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//tools/common/mutate/runner",
    ],
)

go_test(
    name = "mutate_test",
    srcs = [
        "main_test.go",
    ],
    deps = [
        ":mutate",
        "//tools/common/mutate/runner",
    ],
)
