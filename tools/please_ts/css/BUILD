subinclude("///go//build_defs:go")

go_library(
    name = "css",
    srcs = [
        "collision.go",
        "graph.go",
    ],
    visibility = ["//tools/please_ts/..."],
)

go_test(
    name = "css_test",
    srcs = [
        "collision_test.go",
        "graph_test.go",
    ],
    deps = [":css"],
)
