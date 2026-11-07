subinclude("///go//build_defs:go")

go_library(
    name = "kotlin",
    srcs = ["kotlin.go"],
    visibility = ["//tools/..."],
    deps = [
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/kotlin",
    ],
)

go_test(
    name = "kotlin_test",
    srcs = ["kotlin_test.go"],
    deps = [
        ":kotlin",
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/kotlin",
    ],
)
