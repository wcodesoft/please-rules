filegroup(
    name = "rust",
    srcs = ["rust.build_defs"],
    visibility = ["PUBLIC"],
)

subinclude("//build_defs/rust:rust")

rust_toolchain(
    name = "toolchain",
    version = "1.85.0",
    visibility = ["PUBLIC"],
)
