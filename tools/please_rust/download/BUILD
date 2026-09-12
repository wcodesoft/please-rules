subinclude("///go//build_defs:go")

go_library(
    name = "download",
    srcs = [
        "archive.go",
        "download.go",
        "metadata.go",
    ],
    visibility = ["//tools/please_rust/..."],
)

go_test(
    name = "download_test",
    srcs = [
        "archive_test.go",
        "download_test.go",
        "metadata_test.go",
    ],
    deps = [":download"],
)
