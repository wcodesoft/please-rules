subinclude("///go//build_defs:go")

go_library(
    name = "linkorder",
    srcs = ["linkorder.go"],
    visibility = ["//tools/please_swift/..."],
)

go_test(
    name = "linkorder_test",
    srcs = [
        "linkorder.go",
        "linkorder_test.go",
    ],
)
