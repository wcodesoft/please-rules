subinclude("//build_defs/rust:rust")

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
    srcs = ["hermetic_test.rs"],
    crate_name = "hermetic_test",
    deps = [":hermetic_lib"],
)

