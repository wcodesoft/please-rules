subinclude("//build_defs/ts:ts")

ts_library(
    name = "calculator",
    srcs = ["calculator.ts"],
    module_name = "@domain/calculator",
    visibility = ["PUBLIC"],
)

ts_test(
    name = "calculator_test",
    srcs = ["calculator_test.ts"],
    deps = [":calculator"],
)

ts_test(
    name = "calculator_vitest_test",
    srcs = ["calculator_vitest_test.ts"],
    deps = [":calculator"],
    runner = "vitest",
)

