subinclude("//build_defs/ts:ts")

# Pinned npm packages for the fixtures: highlight.js (CommonJS, subpath imports) and
# @codemirror/legacy-modes (ESM with only an exports map, no main/module) and its dependencies.
# Every package is its own target with the sha256 of its tarball; dependencies are listed in deps.
# (Generated from an npm lock; the planned helper would print these declarations.)

ts_npm_module(
    name = "codemirror_language",
    package = "@codemirror/language",
    hashes = ["5e49acf55fde65ce9848068f0f06478be9ec71f818e91de6eca00824e9152226"],
    version = "6.12.4",
    resolve_transitive = False,
    deps = [
        ":codemirror_state",
        ":codemirror_view",
        ":lezer_common",
        ":lezer_highlight",
        ":lezer_lr",
        ":style_mod",
    ],
)

ts_npm_module(
    name = "codemirror_legacy_modes",
    package = "@codemirror/legacy-modes",
    hashes = ["61143cdb9c375dc1521895e4e536d75036baaf6e9df6b4431691f7ea5d273463"],
    version = "6.5.1",
    resolve_transitive = False,
    deps = [
        ":codemirror_language",
    ],
)

ts_npm_module(
    name = "codemirror_state",
    package = "@codemirror/state",
    hashes = ["91d75acc955ceeaf86af9396c11cda7694fe6222aac8427fa033d89eff5ac839"],
    version = "6.7.6",
    resolve_transitive = False,
    deps = [
        ":marijn_find_cluster_break",
    ],
)

ts_npm_module(
    name = "codemirror_view",
    package = "@codemirror/view",
    hashes = ["1246c9b6e1ad1d9d870b3e67672d3a8f8c9bc73af3b63c081c1549a94d5850db"],
    version = "6.43.13",
    resolve_transitive = False,
    deps = [
        ":codemirror_state",
        ":crelt",
        ":style_mod",
        ":w3c_keyname",
    ],
)

ts_npm_module(
    name = "lezer_common",
    package = "@lezer/common",
    hashes = ["a8854639c04adabe5e72c26fa2876475116dc099ae5aa02250a5a938bb5ea4ae"],
    version = "1.5.3",
    resolve_transitive = False,
)

ts_npm_module(
    name = "lezer_highlight",
    package = "@lezer/highlight",
    hashes = ["e349eec1ff4382439c1e6286ae937e0203a8349c5192e2e133863391775ac7ff"],
    version = "1.2.5",
    resolve_transitive = False,
    deps = [
        ":lezer_common",
    ],
)

ts_npm_module(
    name = "lezer_lr",
    package = "@lezer/lr",
    hashes = ["c71e654cd0d153898e744a613bb82714fcd3c7d8e4df8cef7d193fd5ddfa6015"],
    version = "1.4.10",
    resolve_transitive = False,
    deps = [
        ":lezer_common",
    ],
)

ts_npm_module(
    name = "marijn_find_cluster_break",
    package = "@marijn/find-cluster-break",
    hashes = ["5d12b770b64d46632c84fcd18e1d4c09f195ed0f5dbc71f94a41f0af333eaeae"],
    version = "1.0.4",
    resolve_transitive = False,
)

ts_npm_module(
    name = "crelt",
    hashes = ["3542299c0278fdc26aceb2b13d31bffbcb89d7367706bbc457130e3fd103c03e"],
    version = "1.0.7",
    resolve_transitive = False,
)

ts_npm_module(
    name = "highlight_js",
    package = "highlight.js",
    hashes = ["accbfaaab745088609b4eea2bdca2ad62f1f1dd27304e0f8df65cfe0fe042143"],
    version = "11.12.0",
    resolve_transitive = False,
)

ts_npm_module(
    name = "style_mod",
    package = "style-mod",
    hashes = ["769073434ab698eb035cbaa9df5d028231c25d95bc7b8676a46d603fb2e89e39"],
    version = "4.1.4",
    resolve_transitive = False,
)

ts_npm_module(
    name = "w3c_keyname",
    package = "w3c-keyname",
    hashes = ["05d21484026d1f2dd842c843181e391e66d60f1d5dff292d562c3f40302dab36"],
    version = "2.2.8",
    resolve_transitive = False,
)
