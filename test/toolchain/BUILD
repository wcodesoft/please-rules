subinclude("//build_defs/rust:rust")

rust_toolchain(
    name = "tc_185",
    version = "1.85.0",
)

rust_toolchain(
    name = "tc_184",
    version = "1.84.0",
)

rust_toolchain(
    name = "tc_custom_hash",
    version = "1.85.0",
    hashes = {
        "x86_64-unknown-linux-gnu": "be4ba7b777100c851ab268e95f70f405d28d7813ba60a9bdcf4e88c88acf8602",
        "aarch64-unknown-linux-gnu": "0306c30bee00469fbec4b07bb04ea0308c096454354c3dc96a92b729f1c2acd1",
        "x86_64-apple-darwin": "69a36d239e38cc08c6366d1d85071847406645346c6f2d2e0dfaf64b58050d3d",
        "aarch64-apple-darwin": "3ff45cefaf9a002069902acf3a6332113b76b530bb31803fe5cfd30f7ef8ba03",
    },
)

rust_library(
    name = "toolchain_test_lib",
    srcs = ["lib.rs"],
    crate_name = "toolchain_test_lib",
)

rust_bin(
    name = "toolchain_test_bin",
    srcs = ["main.rs"],
    deps = [":toolchain_test_lib"],
)

rust_test(
    name = "toolchain_test_run",
    srcs = ["lib.rs"],
    crate_name = "toolchain_test_lib",
)
