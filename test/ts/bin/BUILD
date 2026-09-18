subinclude("//build_defs/ts:ts")

ts_binary(
    name = "app",
    main = "main.ts",
    deps = ["//test/ts/lib:calculator"],
    flags = ["--allow-read"],
)

ts_bundle(
    name = "app_bundle",
    format = "esm",
    main = "main.ts",
    deps = ["//test/ts/lib:calculator"],
)
