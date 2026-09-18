subinclude("///go//build_defs:go")

go_binary(
    name = "please_wit",
    srcs = [
        "commands.go",
        "main.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//tools/please_wit/generate",
        "//tools/please_wit/toolchain",
    ],
)

go_test(
    name = "main_test",
    srcs = [
        "commands.go",
        "main.go",
        "main_test.go",
    ],
    deps = [
        "//tools/please_wit/generate",
        "//tools/please_wit/toolchain",
    ],
)
