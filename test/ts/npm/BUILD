subinclude("//build_defs/ts:ts")

ts_module(
    name = "clsx",
    hashes = ["6320104b491ca645120e8f99589ce663ff1fb337ba80066719a011bd95320303"],
    package = "clsx",
    version = "2.1.1",
)

ts_library(
    name = "formatter",
    srcs = ["formatter.ts"],
    module_name = "@test/formatter",
    deps = [":clsx"],
)

ts_test(
    name = "formatter_test",
    srcs = ["formatter_test.ts"],
    deps = [
        ":clsx",
        ":formatter",
    ],
)
