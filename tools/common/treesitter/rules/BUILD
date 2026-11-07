subinclude("///go//build_defs:go")

go_library(
    name = "rules",
    srcs = [
        "discover.go",
        "rules.go",
    ],
    visibility = ["//tools/..."],
    deps = ["//third_party/go:gotreesitter"],
)

go_test(
    name = "rules_test",
    srcs = [
        "discover_test.go",
        "rules_test.go",
    ],
    deps = [
        ":rules",
        "//third_party/go:gotreesitter",
        "///third_party/go/github.com_odvcencio_gotreesitter//grammars/go",
    ],
)
