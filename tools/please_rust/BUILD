subinclude("///go//build_defs:go")

go_binary(
    name = "please_rust",
    srcs = [
        "commands.go",
        "main.go",
    ],
    deps = [
        "//tools/please_rust/compile",
        "//tools/please_rust/compilec",
        "//tools/please_rust/download",
        "//tools/please_rust/testrunner",
    ],
    visibility = ["PUBLIC"],
)

go_test(
    name = "main_test",
    srcs = [
        "commands.go",
        "commands_test.go",
        "main.go",
        "main_test.go",
    ],
    deps = [
        "//tools/please_rust/compile",
        "//tools/please_rust/compilec",
        "//tools/please_rust/download",
        "//tools/please_rust/testrunner",
    ],
)

