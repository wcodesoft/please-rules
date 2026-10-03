subinclude("///go//build_defs:go")

go_library(
    name = "lcov",
    srcs = [
        "lcov.go",
        "llvm.go",
        "merge.go",
        "parse.go",
        "path.go",
        "write.go",
    ],
    visibility = ["//tools/..."],
)

go_test(
    name = "lcov_test",
    srcs = [
        "lcov.go",
        "lcov_test.go",
        "llvm.go",
        "llvm_test.go",
        "merge.go",
        "parse.go",
        "path.go",
        "write.go",
    ],
)
