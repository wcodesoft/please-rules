subinclude("//build_defs/ts:ts")

# Standalone minified CSS bundle assembled from transitive css_library deps + page CSS
css_bundle(
    name = "dashboard_styles",
    srcs = ["app.css"],
    out = "dashboard.min.css",
    deps = [
        "//test/ts/css/button",
        "//test/ts/css/card",
    ],
)

# Custom properties set at runtime (here --row-height-*) are allowed explicitly.
css_bundle(
    name = "runtime_vars_styles",
    srcs = ["runtime_vars.css"],
    out = "runtime_vars.min.css",
    allow_undefined_vars = ["--row-height-*"],
    deps = ["//test/ts/css/tokens"],
)

gentest(
    name = "css_bundle_test",
    data = {
        "css": [":dashboard_styles"],
        "please_ts": ["//tools/please_ts"],
        "button": ["//test/ts/css/button"],
        "colliding": ["collision/colliding.css"],
        "undefined": ["undefined_var/undefined.css"],
        "runtime": [":runtime_vars_styles"],
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
        'if $DATA_PLEASE_TS css bundle --out collision_out.css --srcs "$DATA_COLLIDING" --deps "$DATA_BUTTON" 2>collision.err; then exit 1; fi',
        'grep -q \'duplicate CSS selector ".btn"\' collision.err',
        'grep -q "button.css:2" collision.err',
        'grep -q "colliding.css:2" collision.err',
        # A custom property used without a fallback and defined nowhere fails the build, naming the
        # property and the line; a use with a fallback, and one defined in a dependency, do not.
        'if $DATA_PLEASE_TS css bundle --out vars_out.css --srcs "$DATA_UNDEFINED" --deps "$DATA_BUTTON" 2>vars.err; then exit 1; fi',
        'grep -q "undefined-variable check failed" vars.err',
        'grep -q -- "--brand-missing: 1 use without a fallback" vars.err',
        'grep -q "undefined.css:2" vars.err',
        '! grep -q -- "--soft-missing" vars.err',
        '! grep -q -- "--color-brand" vars.err',
        # Naming the property in --allow-undefined-vars lifts the check (esbuild is not on PATH here,
        # so the command may still fail afterwards, but not on this check).
        '$DATA_PLEASE_TS css bundle --out vars_ok.css --srcs "$DATA_UNDEFINED" --deps "$DATA_BUTTON" --allow-undefined-vars "--brand-*" 2>vars_ok.err || true',
        '! grep -q "undefined-variable check failed" vars_ok.err',
        # The css_bundle rule passes allow_undefined_vars through: the target built, with the variable kept.
        'grep -q -- "--row-height-compact" $DATA_RUNTIME',
    ]),
)
