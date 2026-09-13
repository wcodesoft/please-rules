subinclude("///go//build_defs:go")

go_library(
    name = "compile",
    srcs = [
        "compile.go",
        "jar.go",
    ],
    deps = [
        "//tools/please_kotlin/toolchain",
    ],
    visibility = ["//tools/please_kotlin/..."],
)

go_test(
    name = "compile_test",
    srcs = [
        "compile_test.go",
        "jar_test.go",
    ],
    deps = [
        ":compile",
    ],
)
