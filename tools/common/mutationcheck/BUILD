subinclude("///go//build_defs:go")

go_binary(
    name = "mutationcheck",
    srcs = ["main.go"],
    visibility = ["PUBLIC"],
    deps = ["//tools/common/mutation"],
)

go_test(
    name = "mutationcheck_test",
    srcs = ["main_test.go"],
    deps = [
        ":mutationcheck",
        "//tools/common/mutation",
    ],
)
