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

# The 800px viewport keeps the max-width: 700px media query off, so the golden holds the wide
# layout on every machine.
ts_browser_test(
    name = "snapshot_browser_test",
    srcs = ["snapshot_browser_test.ts"],
    snapshots = ["snapshots/card.json"],
    viewport = "800x600",
)
