subinclude("///go//build_defs:go")

go_binary(
    name = "please_ts",
    srcs = [
        "commands.go",
        "main.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//tools/please_ts/binary",
        "//tools/please_ts/bundle",
        "//tools/please_ts/compile",
        "//tools/please_ts/importmap",
        "//tools/please_ts/testrunner",
        "//tools/please_ts/unpack",
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
        "//tools/please_ts/binary",
        "//tools/please_ts/bundle",
        "//tools/please_ts/compile",
        "//tools/please_ts/importmap",
        "//tools/please_ts/testrunner",
        "//tools/please_ts/unpack",
    ],
)
