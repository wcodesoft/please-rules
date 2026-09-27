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

ts_module(
    name = "debug",
    hashes = ["fd9c4f0ca9acd1103099f038754eb21f94497456fb9b195b18fd582fa6982d1c"],
    package = "debug",
    version = "4.4.0",
)

ts_module(
    name = "ms",
    hashes = ["f6616e15e530ed552f9daa2d3ce71963947c6bc7c98c9b64fd3e673fd02622c6"],
    package = "ms",
    version = "2.1.3",
)

ts_library(
    name = "logger",
    srcs = ["logger.ts"],
    module_name = "@test/logger",
    deps = [":debug"],
)

ts_test(
    name = "logger_test",
    srcs = ["logger_test.ts"],
    deps = [
        ":debug",
        ":logger",
        ":ms",
    ],
)
