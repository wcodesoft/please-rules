subinclude("///go//build_defs:go")

go_library(
    name = "go",
    srcs = ["go.go"],
    visibility = ["//tools/..."],
    deps = [
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/go",
    ],
)

go_test(
    name = "go_test",
    srcs = ["go_test.go"],
    deps = [
        ":go",
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/go",
    ],
)
