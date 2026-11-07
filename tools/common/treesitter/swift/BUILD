subinclude("///go//build_defs:go")

go_library(
    name = "swift",
    srcs = ["swift.go"],
    visibility = ["//tools/..."],
    deps = [
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/swift",
    ],
)

go_test(
    name = "swift_test",
    srcs = ["swift_test.go"],
    deps = [
        ":swift",
        "//tools/common/treesitter/rules",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/swift",
    ],
)
