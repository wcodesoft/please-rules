subinclude("//build_defs/ts:ts")

# Regression tests for wcodesoft/please-rules#107: a test stages the outputs of its direct
# dependencies only, so a library's own dependencies (`lib`, `routes`, and `base` below `lib`) must
# travel inside the library's output. The tests depend on `components` alone.

ts_browser_test(
    name = "transitive_browser_test",
    srcs = ["transitive_browser_test.ts"],
    deps = ["//test/ts/transitive_deps/components"],
)

ts_test(
    name = "transitive_deno_test",
    srcs = ["transitive_deno_test.ts"],
    deps = ["//test/ts/transitive_deps/components"],
)

# The same browser test with the transitive dependencies also listed directly (the workaround).
# This one passes and stays in the default run, so the workaround keeps working.
ts_browser_test(
    name = "listed_directly_browser_test",
    srcs = ["transitive_browser_test.ts"],
    deps = [
        "//test/ts/transitive_deps/components",
        "//test/ts/transitive_deps/lib",
        "//test/ts/transitive_deps/routes",
    ],
)

# A first-party library whose own import is a third-party npm module (`clsx`); the test lists only
# the library.
ts_browser_test(
    name = "third_party_transitive_browser_test",
    srcs = ["third_party_transitive_test.ts"],
    deps = ["//test/ts/transitive_deps/styled"],
)

ts_test(
    name = "third_party_transitive_deno_test",
    srcs = ["third_party_transitive_test.ts"],
    deps = ["//test/ts/transitive_deps/styled"],
)
