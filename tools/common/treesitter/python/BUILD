subinclude("///go//build_defs:go")

go_library(
    name = "python",
    srcs = ["python.go"],
    visibility = ["//tools/..."],
    deps = [
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/python",
    ],
)

go_test(
    name = "python_test",
    srcs = ["python_test.go"],
    deps = [
        ":python",
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/python",
    ],
)
