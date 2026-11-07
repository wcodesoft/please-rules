subinclude("//build_defs/ts:ts")

# A first-party library that imports a third-party npm module.
ts_library(
    name = "styled",
    srcs = ["styled.ts"],
    module_name = "@transitive/styled",
    deps = ["//test/ts/npm:clsx"],
    visibility = ["PUBLIC"],
)
