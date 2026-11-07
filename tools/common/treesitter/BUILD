subinclude("///go//build_defs:go")

go_library(
    name = "treesitter",
    srcs = ["treesitter.go"],
    visibility = ["//tools/..."],
    deps = [
        "//third_party/go:gotreesitter",
        "//tools/common/treesitter/go",
        "//tools/common/treesitter/kotlin",
        "//tools/common/treesitter/python",
        "//tools/common/treesitter/rules",
        "//tools/common/treesitter/rust",
        "//tools/common/treesitter/swift",
        "//tools/common/treesitter/typescript",
    ],
)

go_test(
    name = "treesitter_test",
    srcs = ["treesitter_test.go"],
    deps = [
        ":treesitter",
        "//third_party/go:gotreesitter",
        "//tools/common/treesitter/go",
        "//tools/common/treesitter/kotlin",
        "//tools/common/treesitter/python",
        "//tools/common/treesitter/rules",
        "//tools/common/treesitter/rust",
        "//tools/common/treesitter/swift",
        "//tools/common/treesitter/typescript",
    ],
)
