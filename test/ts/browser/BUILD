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

# Colored boxes only, so the goldens do not depend on the fonts of the machine that renders them.
ts_browser_test(
    name = "screenshot_browser_test",
    srcs = ["screenshot_browser_test.ts"],
    screenshots = [
        "screenshots/card.png",
        "screenshots/page.png",
    ],
    viewport = "800x600",
)
