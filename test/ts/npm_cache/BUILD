subinclude("//build_defs/ts:ts")

# A CommonJS package and its transitive dependency, each pinned by the sha256 of its tarball.
ts_npm_module(
    name = "ms",
    hashes = ["f6616e15e530ed552f9daa2d3ce71963947c6bc7c98c9b64fd3e673fd02622c6"],
    version = "2.1.3",
    resolve_transitive = False,
)

ts_npm_module(
    name = "debug",
    hashes = ["c803a8ca9b835b7a75f7150ee52f7f640675515bafbe8f5da78fcc0ae12914ac"],
    version = "4.3.7",
    resolve_transitive = False,
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

# Automatic resolution: only the root tarball is pinned. The dependencies are resolved from
# the registry when the module is built, as of the day the root version was published, and
# verified against the registry's integrity data.
ts_npm_module(
    name = "debug_auto",
    package = "debug",
    hashes = ["c803a8ca9b835b7a75f7150ee52f7f640675515bafbe8f5da78fcc0ae12914ac"],
    version = "4.3.7",
)

ts_library(
    name = "humanize_auto",
    srcs = ["humanize.ts"],
    module_name = "@test/humanize_auto",
    deps = [":debug_auto"],
)

ts_test(
    name = "humanize_auto_test",
    srcs = ["humanize_auto_test.ts"],
    deps = [
        ":debug_auto",
        ":humanize_auto",
    ],
)

# The same exports-only package as modes_test, with its eleven dependencies resolved
# automatically instead of declared.
ts_npm_module(
    name = "legacy_modes_auto",
    package = "@codemirror/legacy-modes",
    hashes = ["61143cdb9c375dc1521895e4e536d75036baaf6e9df6b4431691f7ea5d273463"],
    version = "6.5.1",
)

ts_npm_module(
    name = "codemirror_language_auto",
    package = "@codemirror/language",
    hashes = ["5e49acf55fde65ce9848068f0f06478be9ec71f818e91de6eca00824e9152226"],
    version = "6.12.4",
)

ts_library(
    name = "modes_auto",
    srcs = ["modes.ts"],
    module_name = "@test/modes_auto",
    deps = [
        ":codemirror_language_auto",
        ":legacy_modes_auto",
    ],
)

ts_test(
    name = "modes_auto_test",
    srcs = ["modes_auto_test.ts"],
    deps = [
        ":codemirror_language_auto",
        ":legacy_modes_auto",
        ":modes_auto",
    ],
)

# Bundling and compiling with npm packages: Deno resolves them from the cache slices (no network,
# no node_modules), and `deno bundle` runs the pinned esbuild.
ts_bundle(
    name = "highlight_bundle",
    srcs = ["highlight_main.ts"],
    format = "iife",
    main = "highlight_main.ts",
    deps = [
        ":highlight",
        "//test/ts/npm_cache/modules:highlight_js",
    ],
)

ts_binary(
    name = "highlight_app",
    srcs = ["highlight_main.ts"],
    main = "highlight_main.ts",
    deps = [
        ":highlight",
        "//test/ts/npm_cache/modules:highlight_js",
    ],
)

# The exports-only package with its declared closure, and with the closure resolved automatically.
ts_bundle(
    name = "modes_bundle",
    srcs = ["modes.ts"],
    format = "iife",
    main = "modes.ts",
    deps = [
        "//test/ts/npm_cache/modules:codemirror_language",
        "//test/ts/npm_cache/modules:codemirror_legacy_modes",
    ],
)

ts_bundle(
    name = "modes_auto_bundle",
    srcs = ["modes.ts"],
    format = "iife",
    main = "modes.ts",
    deps = [
        ":codemirror_language_auto",
        ":legacy_modes_auto",
    ],
)

gentest(
    name = "bundles_test",
    data = {
        "highlight": [":highlight_bundle"],
        "modes": [":modes_bundle"],
        "modes_auto": [":modes_auto_bundle"],
        "app": [":highlight_app"],
    },
    no_test_output = True,
    test_cmd = " && ".join([
        # self-contained: the npm packages are inlined, no specifier is left to resolve at run time
        '! grep -q "npm:" $DATA_HIGHLIGHT',
        '! grep -q "highlight.js/lib" $DATA_HIGHLIGHT',
        # ...and their code really is in there: highlight.js's Go grammar, CodeMirror's StreamLanguage
        'grep -q "registerLanguage" $DATA_HIGHLIGHT',
        'grep -q "golang" $DATA_HIGHLIGHT',
        '! grep -q "npm:" $DATA_MODES',
        'grep -q "StreamLanguage" $DATA_MODES',
        'grep -q "StreamLanguage" $DATA_MODES_AUTO',
        # the compiled program carries highlight.js and runs without any cache or network
        'out=$(DENO_DIR=/nonexistent $DATA_APP)',
        'echo "$out" | grep -q "hljs-keyword"',
    ]),
)
