subinclude("//build_defs/ts:ts")

ts_library(
    name = "counter_component",
    srcs = ["counter_component.ts"],
    module_name = "@test/browser/counter",
    visibility = ["PUBLIC"],
)

ts_browser_test(
    name = "counter_browser_test",
    srcs = ["counter_browser_test.ts"],
    deps = [":counter_component"],
)
