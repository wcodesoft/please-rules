subinclude("//build_defs/rust:rust")

rust_crate(
    name = "itoa",
    version = "1.0.10",
)

rust_bin(
    name = "crate_test_bin",
    srcs = ["main.rs"],
    deps = [":itoa"],
)
