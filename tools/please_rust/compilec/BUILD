subinclude("///go//build_defs:go")

go_library(
    name = "compilec",
    srcs = ["compile_c.go"],
    visibility = ["//tools/please_rust/..."],
)

go_test(
    name = "compilec_test",
    srcs = ["compile_c_test.go"],
    deps = [":compilec"],
)

