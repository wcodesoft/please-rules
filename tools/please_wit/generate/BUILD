subinclude("///go//build_defs:go")

go_library(
    name = "generate",
    srcs = ["generate.go"],
    visibility = ["//tools/please_wit/..."],
)

go_test(
    name = "generate_test",
    srcs = [
        "generate.go",
        "generate_test.go",
    ],
)
