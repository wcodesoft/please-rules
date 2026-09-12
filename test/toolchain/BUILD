subinclude("//build_defs/rust:rust")

rust_toolchain(
    name = "toolchain",
    version = "1.85.0",
)

rust_library(
    name = "hermetic_lib",
    srcs = ["lib.rs"],
    crate_name = "hermetic_lib",
)

rust_bin(
    name = "hermetic_bin",
    srcs = ["main.rs"],
    deps = [":hermetic_lib"],
)

rust_test(
    name = "hermetic_test",
    srcs = ["lib.rs"],
    crate_name = "hermetic_lib",
)

