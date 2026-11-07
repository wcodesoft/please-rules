subinclude("///go//build_defs:go")

go_library(
    name = "mutation",
    srcs = [
        "mutation.go",
        "path.go",
        "report.go",
    ],
    visibility = ["//tools/..."],
)

go_test(
    name = "mutation_test",
    srcs = [
        "mutation_test.go",
        "path_test.go",
        "report_test.go",
    ],
    deps = [":mutation"],
)
