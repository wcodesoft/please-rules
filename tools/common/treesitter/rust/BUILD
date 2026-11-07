subinclude("///go//build_defs:go")

go_library(
    name = "rust",
    srcs = ["rust.go"],
    visibility = ["//tools/..."],
    deps = [
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/rust",
    ],
)

go_test(
    name = "rust_test",
    srcs = ["rust_test.go"],
    deps = [
        ":rust",
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/rust",
    ],
)
