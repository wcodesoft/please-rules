subinclude("//build_defs/ts:ts")

# A CommonJS package and its transitive dependency, each pinned by the sha256 of its tarball.
ts_npm_module(
    name = "ms",
    hashes = ["f6616e15e530ed552f9daa2d3ce71963947c6bc7c98c9b64fd3e673fd02622c6"],
    version = "2.1.3",
)

ts_npm_module(
    name = "debug",
    hashes = ["c803a8ca9b835b7a75f7150ee52f7f640675515bafbe8f5da78fcc0ae12914ac"],
    version = "4.3.7",
    deps = [":ms"],
)

ts_library(
    name = "humanize",
    srcs = ["humanize.ts"],
    module_name = "@test/humanize",
    deps = [":debug"],
)

ts_test(
    name = "humanize_test",
    srcs = ["humanize_test.ts"],
    deps = [
        ":debug",
        ":humanize",
    ],
)

# highlight.js is CommonJS and is imported through subpaths (highlight.js/lib/core).
ts_library(
    name = "highlight",
    srcs = ["highlight.ts"],
    module_name = "@test/highlight",
    deps = ["//test/ts/npm_cache/modules:highlight_js"],
)

ts_test(
    name = "highlight_test",
    srcs = ["highlight_test.ts"],
    deps = [
        ":highlight",
        "//test/ts/npm_cache/modules:highlight_js",
    ],
)

# @codemirror/legacy-modes is ESM with only an exports map (./mode/*) and no main or module.
ts_library(
    name = "modes",
    srcs = ["modes.ts"],
    module_name = "@test/modes",
    deps = [
        "//test/ts/npm_cache/modules:codemirror_language",
        "//test/ts/npm_cache/modules:codemirror_legacy_modes",
    ],
)

ts_test(
    name = "modes_test",
    srcs = ["modes_test.ts"],
    deps = [
        ":modes",
        "//test/ts/npm_cache/modules:codemirror_language",
        "//test/ts/npm_cache/modules:codemirror_legacy_modes",
    ],
)
