subinclude("//build_defs/ts:ts")

# A module whose module_name is the exact import (no sub-path): `import ... from "@transitive/routes"`.
ts_library(
    name = "routes",
    srcs = ["router.ts"],
    module_name = "@transitive/routes",
    visibility = ["PUBLIC"],
)
