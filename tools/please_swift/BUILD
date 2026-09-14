subinclude("///go//build_defs:go")

go_binary(
    name = "please_swift",
    srcs = [
        "commands.go",
        "main.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//tools/please_swift/binary",
        "//tools/please_swift/compile",
        "//tools/please_swift/testrunner",
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
        "//tools/please_swift/binary",
        "//tools/please_swift/compile",
        "//tools/please_swift/testrunner",
    ],
)
