subinclude("///go//build_defs:go")

go_library(
    name = "ast",
    srcs = [
        "ast.go",
        "parser.go",
    ],
    visibility = ["//tools/please_wit/..."],
)

go_test(
    name = "ast_test",
    srcs = [
        "ast_test.go",
        "parser_test.go",
    ],
    deps = [
        ":ast",
    ],
)
