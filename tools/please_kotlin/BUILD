subinclude("///go//build_defs:go")

go_binary(
    name = "please_kotlin",
    srcs = [
        "commands.go",
        "main.go",
    ],
    deps = [
        "//tools/please_kotlin/compile",
        "//tools/please_kotlin/testrunner",
        "//tools/please_kotlin/toolchain",
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
        "//tools/please_kotlin/compile",
        "//tools/please_kotlin/testrunner",
        "//tools/please_kotlin/toolchain",
    ],
)
