subinclude("//build_defs/ts:ts")

# 1. Standalone minified CSS bundle assembled from transitive ts_library deps + page CSS
css_bundle(
    name = "dashboard_styles",
    srcs = ["app.css"],
    out = "dashboard.min.css",
    deps = [
        "//test/ts/css/button",
        "//test/ts/css/card",
    ],
)

# 2. Combined JS + CSS bundle from a single ts_bundle target
ts_bundle(
    name = "dashboard_app",
    srcs = [
        "app.css",
        "app.ts",
    ],
    css_out = "dashboard_app.css",
    main = "app.ts",
    minify = True,
    out = "dashboard_app.js",
    deps = ["//test/ts/css/card"],
)

gentest(
    name = "css_bundle_test",
    data = {
        "css": [":dashboard_styles"],
        "app": [":dashboard_app"],
        "please_ts": ["//tools/please_ts"],
        "button": ["//test/ts/css/button"],
        "colliding": ["collision/colliding.css"],
    },
    no_test_output = True,
    test_cmd = " && ".join([
        # Minified: no comments, hex shortened (#000000 -> #000), margin shorthand collapsed (margin:4px 8px)
        '! grep -q "Design tokens" $DATA_CSS',
        'grep -q -- "--color-brand: #ffffff" $DATA_CSS',
        'grep -q "margin:4px 8px" $DATA_CSS',
        'grep -q "background-color:#000" $DATA_CSS',
        'grep -q ".dashboard{display:flex" $DATA_CSS',
        # Diamond dependency (tokens.css) is included exactly once
        'test $(grep -o -- "--color-brand" $DATA_CSS | wc -l) -eq 2',
        # Topological order: tokens (:root) appear before .btn and .card
        'awk \'index($0,":root") < index($0,".btn") && index($0,".btn") < index($0,".dashboard") {ok=1} END {exit !ok}\' $DATA_CSS',
        # Cross-file collision check fails loudly naming both files and line numbers
        'if $DATA_PLEASE_TS bundle --out collision_out.css --srcs "$DATA_COLLIDING" --deps "$DATA_BUTTON" 2>collision.err; then exit 1; fi',
        'grep -q \'duplicate CSS selector ".btn"\' collision.err',
        'grep -q "button.css:2" collision.err',
        'grep -q "colliding.css:2" collision.err',
    ]),
)

ts_browser_test(
    name = "css_browser_test",
    srcs = ["css_browser_test.ts"],
     data = ["app.css"],
    deps = ["//test/ts/css/card"],
)
