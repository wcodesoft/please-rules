subinclude("//build_defs/ts:ts")

ts_library(
    name = "app",
    srcs = ["app.ts"],
    module_name = "@scope/app",
    visibility = ["PUBLIC"],
    deps = ["//test/ts/nested/widget"],
)

ts_test(
    name = "app_test",
    srcs = ["app_test.ts"],
    runner = "vitest",
    deps = [
        ":app",
        "//test/ts/nested/widget",
    ],
)
