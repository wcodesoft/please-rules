subinclude("//build_defs/ts:ts")

# Regression tests for wcodesoft/please-rules#107: a test that depends on `components` only does not
# get the files of the libraries `components` imports (`lib`, `routes`), because a test stages the
# outputs of its direct dependencies and `data` is not transitive. The two targets that show it are
# labelled `manual` so `//...` stays green; drop the label together with the fix.

ts_browser_test(
    name = "transitive_browser_test",
    srcs = ["transitive_browser_test.ts"],
    deps = ["//test/ts/transitive_deps/components"],
    labels = ["manual", "ts", "test", "browser"],
)

ts_test(
    name = "transitive_deno_test",
    srcs = ["transitive_deno_test.ts"],
    deps = ["//test/ts/transitive_deps/components"],
    labels = ["manual", "ts", "test"],
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
