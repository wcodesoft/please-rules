subinclude("///go//build_defs:go")

go_binary(
    name = "please_rust",
    srcs = ["main.go"],
    deps = [
        "//tools/please_rust/compile",
        "//tools/please_rust/compilec",
        "//tools/please_rust/download",
        "//tools/please_rust/testrunner",
    ],
    visibility = ["PUBLIC"],
)
