subinclude("///go//build_defs:go")

go_library(
    name = "typescript",
    srcs = ["typescript.go"],
    visibility = ["//tools/..."],
    deps = [
        "//third_party/go:gotreesitter",
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/javascript",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/tsx",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/typescript",
    ],
)

go_test(
    name = "typescript_test",
    srcs = ["typescript_test.go"],
    deps = [
        ":typescript",
        "//third_party/go:gotreesitter",
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/javascript",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/tsx",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/typescript",
    ],
)
