subinclude("//build_defs/rust:rust")

rust_library(
    name = "lib",
    srcs = ["lib.rs"],
    crate_name = "mylib",
    visibility = ["PUBLIC"],
)

rust_test(
    name = "lib_test",
    srcs = ["lib_test.rs"],
    crate_name = "mylib_test",
    deps = [":lib"],
)

