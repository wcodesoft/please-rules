subinclude("//build_defs/ts:ts")

# `lib` imports `base`: a test that depends on `components` reaches `base` three levels down.
ts_library(
    name = "lib",
    srcs = ["greeting.ts"],
    module_name = "@transitive/lib",
    deps = ["//test/ts/transitive_deps/base"],
    visibility = ["PUBLIC"],
)
