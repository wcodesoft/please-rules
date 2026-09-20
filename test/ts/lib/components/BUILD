subinclude("//build_defs/ts:ts")

# 1. Multi-component library with explicit module_name:
ts_library(
    name = "components",
    srcs = [
        "Badge.ts",
        "MetricCard.ts",
        "Modal.ts",
    ],
    module_name = "@repo/dashboard/components",
    visibility = ["PUBLIC"],
)

# 2. Multi-component library with auto-derived module_name:
ts_library(
    name = "auto_named",
    srcs = [
        "Badge.ts",
        "MetricCard.ts",
    ],
    visibility = ["PUBLIC"],
)

# 3. Test verifying direct component imports without barrel index.ts:
ts_test(
    name = "components_test",
    srcs = ["components_test.ts"],
    deps = [
        ":components",
        ":auto_named",
    ],
)
